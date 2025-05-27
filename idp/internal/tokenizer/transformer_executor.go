package tokenizer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/policy"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/ucv8go"
)

// ExecuteTransformerParameters is the request to execute a specific transformer on the passed Data
type ExecuteTransformerParameters struct {
	Transformer         *storage.Transformer
	TokenAccessPolicyID uuid.UUID
	Data                string
	DataProvenance      *policy.UserstoreDataProvenance
}

// TransformerExecutor is used to perform transformations for a series of ExecuteTransformerParameters.
// It maintains a cache of initialized transformer handlers, so that startup for each distinct transformer
// only happens once for the life of the TransformerExecutor. The caller must call CleanupExecution to
// free resources that were created during the life of the TransformerExecutor.
type TransformerExecutor struct {
	s                            *storage.Storage
	authzClient                  *authz.Client
	handlersByTransformerAndAPID map[string]*transformerHandler
}

// NewTransformerExecutor creates a TransformerExecutor
func NewTransformerExecutor(
	s *storage.Storage,
	authzClient *authz.Client,
) TransformerExecutor {
	return TransformerExecutor{
		s:                            s,
		authzClient:                  authzClient,
		handlersByTransformerAndAPID: map[string]*transformerHandler{},
	}
}

// CleanupExecution will clean up the execution of all underlying transformer handlers
func (te *TransformerExecutor) CleanupExecution() {
	for _, th := range te.handlersByTransformerAndAPID {
		th.reset()
		th.cleanupExecution()
	}
}

// Execute performs transformations for a set of ExecuteTransformerParameters.
func (te *TransformerExecutor) Execute(
	ctx context.Context,
	tps ...ExecuteTransformerParameters,
) (results []string, consoleOutput string, err error) {
	// reset transformer handlers

	te.reset()

	// group transformer parameters

	transformerHandlers := map[string]*transformerHandler{}
	resultIndicesByTransformerID := map[string][]int{}
	for i, tp := range tps {
		transformerAndAPID := fmt.Sprintf("%v-%v", tp.Transformer.ID, tp.TokenAccessPolicyID)
		th := te.handlersByTransformerAndAPID[transformerAndAPID]
		if th == nil {
			th, err = newTransformerHandler(ctx, te.s, te.authzClient, tp.Transformer, tp.TokenAccessPolicyID)
			if err != nil {
				return nil, "", ucerr.Wrap(err)
			}
			te.handlersByTransformerAndAPID[transformerAndAPID] = th
		}
		transformerHandlers[transformerAndAPID] = th

		if err = th.addData(tp.Data, tp.DataProvenance); err != nil {
			return nil, "", ucerr.Wrap(err)
		}
		resultIndices := resultIndicesByTransformerID[transformerAndAPID]
		resultIndices = append(resultIndices, i)
		resultIndicesByTransformerID[transformerAndAPID] = resultIndices
	}

	// execute transformers and partition the results

	resultsByIndex := map[int]string{}
	consoleOutput = ""
	for key, th := range transformerHandlers {
		results, err := th.execute(ctx)
		if err != nil {
			return nil, "", ucerr.Wrap(err)
		}

		for i, index := range resultIndicesByTransformerID[key] {
			resultsByIndex[index] = results[i]
		}
		consoleOutput += th.getConsoleOutput()
	}

	results = make([]string, len(resultsByIndex))
	for i, result := range resultsByIndex {
		results[i] = result
	}

	return results, consoleOutput, nil
}

func (te *TransformerExecutor) reset() {
	for _, th := range te.handlersByTransformerAndAPID {
		th.reset()
	}
}

type transformerHandler struct {
	s                 *storage.Storage
	authzClient       *authz.Client
	transformer       *storage.Transformer
	nativeFunction    *storage.TransformerFunc
	jsContext         *ucv8go.Context
	jsScript          string
	tokenAccessPolicy *storage.AccessPolicy
	consoleBuilder    *strings.Builder
	data              []string
	dataProvenance    []*policy.UserstoreDataProvenance
	setupComplete     bool
}

func newTransformerHandler(
	ctx context.Context,
	s *storage.Storage,
	authzClient *authz.Client,
	transformer *storage.Transformer,
	tokenAccessPolicyID uuid.UUID,
) (*transformerHandler, error) {
	if s == nil {
		return nil, ucerr.New("Storage cannot be nil")
	}

	if authzClient == nil {
		return nil, ucerr.New("authz.Client cannot be nil")
	}

	if transformer == nil {
		return nil, ucerr.New("Transformer cannot be nil")
	}

	if err := transformer.Validate(); err != nil {
		return nil, ucerr.Wrap(err)
	}

	th := transformerHandler{
		s:              s,
		authzClient:    authzClient,
		transformer:    transformer,
		nativeFunction: storage.GetNativeTransformer(transformer.ID),
		consoleBuilder: &strings.Builder{},
	}

	if transformer.RequiresTokenAccessPolicy() {
		if tokenAccessPolicyID.IsNil() {
			return nil, ucerr.New("tokenAccessPolicy cannot be nil for tokenizing transfomer")
		}

		latestAccessPolicy, err := s.GetLatestAccessPolicy(ctx, tokenAccessPolicyID)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		th.tokenAccessPolicy = latestAccessPolicy
	}

	return &th, nil
}

func (th *transformerHandler) addData(
	data string,
	dataProvenance *policy.UserstoreDataProvenance,
) error {
	if th.transformer.RequiresDataProvenance() {
		if dataProvenance == nil {
			return ucerr.New("dataProvenance must be specified")
		}
		if err := dataProvenance.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}

	th.data = append(th.data, data)
	th.dataProvenance = append(th.dataProvenance, dataProvenance)

	return nil
}

func (th *transformerHandler) cleanupExecution() {
	th.reset()
	if th.jsContext != nil {
		res := th.jsContext.Res
		th.jsContext.Close()
		res.Release()
		th.jsContext = nil
		th.jsScript = ""
	}
	th.setupComplete = false
}

func (transformerHandler) encodeData(data string) (string, error) {
	// TODO: there's probably many better ways to do this, but we want to accept either
	// a string, or a JSON-encoded piece of data (dict, array, string seems reasonable to cover that?)
	// TODO: this should probably also live somewhere it can be used in eg Validate?
	dataBytes := []byte(data)
	encodedData := data
	testMap := map[string]any{}
	testArray := []any{}
	var testString string
	if json.Unmarshal(dataBytes, &testMap) != nil &&
		json.Unmarshal(dataBytes, &testArray) != nil &&
		json.Unmarshal(dataBytes, &testString) != nil {
		// failed to decode this as map, array, or string
		// so we'll encode it ourselves
		bs, err := json.Marshal(data)
		if err != nil {
			// this error is actually hard to recover from
			return "", ucerr.Wrap(err)
		}
		encodedData = string(bs)
	}
	return encodedData, nil
}

func (th *transformerHandler) execute(ctx context.Context) ([]string, error) {
	// request may contain PII so don't log it in prod
	uclog.DebugfPII(ctx, "executing transformer %v", th.transformer.ID)

	start := time.Now().UTC()
	defer logTransformerDuration(ctx, th.transformer.ID, th.transformer.Version, start)
	logTransformerCall(ctx, th.transformer.ID, th.transformer.Version)

	transformedData, untransformedIndices, err := th.lookup(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if untransformedIndices.Size() > 0 {
		if err := th.setupExecution(ctx); err != nil {
			return nil, ucerr.Wrap(err)
		}

		for numTries := 1; untransformedIndices.Size() > 0 && numTries <= maxTokenUniquenessTries; numTries++ {
			remainingIndices := set.NewIntSet()
			for _, index := range untransformedIndices.Items() {
				// TODO: once we support token deletion, we should clean up any tokens that were
				//       created and saved as part of this request if we encounter an unrecoverable
				//       error
				candidateData, err := th.transform(ctx, index)
				if err != nil {
					return nil, ucerr.Wrap(err)
				}

				isUnique, err := th.save(ctx, index, candidateData, numTries)
				if err != nil {
					return nil, ucerr.Wrap(err)
				}

				if isUnique {
					transformedData[index] = candidateData
				} else {
					remainingIndices.Insert(index)
				}
			}
			untransformedIndices = remainingIndices
		}

		if untransformedIndices.Size() > 0 {
			return nil,
				ucerr.Friendlyf(
					nil,
					"exceeded %d attempts to transform values for transformer %v",
					maxTokenUniquenessTries,
					th.transformer.ID,
				)
		}
	}

	uclog.Verbosef(ctx, "got results: %+v", transformedData)

	return transformedData, nil
}

func (th *transformerHandler) getConsoleOutput() string {
	return th.consoleBuilder.String()
}

func (th transformerHandler) lookup(
	ctx context.Context,
) ([]string, set.Set[int], error) {
	transformedData := make([]string, len(th.data))
	untransformedIndices := set.NewIntSet()

	for i := range th.data {
		var trs []storage.TokenRecord

		if th.transformer.ReuseExistingToken {
			var err error

			switch th.transformer.TransformType.ToClient() {
			case policy.TransformTypeTokenizeByValue:
				trs, err = th.s.ListTokenRecordsByDataAndPolicy(
					ctx,
					th.data[i],
					th.transformer.ID,
					th.tokenAccessPolicy.ID,
				)
			case policy.TransformTypeTokenizeByReference:
				trs, err = th.s.ListTokenRecordsByDataProvenanceAndPolicy(
					ctx,
					th.dataProvenance[i].UserID,
					th.dataProvenance[i].ColumnID,
					th.transformer.ID,
					th.tokenAccessPolicy.ID,
				)
			}

			if err != nil {
				return nil, untransformedIndices, ucerr.Wrap(err)
			}
		}

		if len(trs) > 0 {
			transformedData[i] = trs[0].Token
		} else {
			untransformedIndices.Insert(i)
		}
	}

	return transformedData, untransformedIndices, nil
}

func (th *transformerHandler) reset() {
	th.data = []string{}
	th.dataProvenance = []*policy.UserstoreDataProvenance{}
	th.consoleBuilder.Reset()
}

func (th transformerHandler) save(
	ctx context.Context,
	index int,
	transformedData string,
	numTries int,
) (bool, error) {
	var tr *storage.TokenRecord

	switch th.transformer.TransformType.ToClient() {
	case policy.TransformTypeTokenizeByValue:
		tr = &storage.TokenRecord{
			BaseModel:          ucdb.NewBase(),
			TransformerID:      th.transformer.ID,
			TransformerVersion: th.transformer.Version,
			AccessPolicyID:     th.tokenAccessPolicy.ID,
			Token:              transformedData,
			Data:               th.data[index],
		}
	case policy.TransformTypeTokenizeByReference:
		tr = &storage.TokenRecord{
			BaseModel:          ucdb.NewBase(),
			TransformerID:      th.transformer.ID,
			TransformerVersion: th.transformer.Version,
			AccessPolicyID:     th.tokenAccessPolicy.ID,
			Token:              transformedData,
			UserID:             th.dataProvenance[index].UserID,
			ColumnID:           th.dataProvenance[index].ColumnID,
		}
	default:
		return true, nil
	}

	if err := th.s.SaveTokenRecord(ctx, tr); err != nil {
		if !ucdb.IsUniqueViolation(err) {
			return false, ucerr.Wrap(err)
		}

		uclog.Warningf(
			ctx, "transformer %v created duplicate token value %v: %v",
			th.transformer.ID,
			tr.Token,
			err,
		)

		if numTries >= maxTokenUniquenessTries {
			logTransformerConflict(ctx, th.transformer.ID, th.transformer.Version)
			return false,
				ucerr.Friendlyf(
					err,
					"exceeded %d attempts to generate non-conflicting tokens for transformer %v",
					maxTokenUniquenessTries,
					th.transformer.ID,
				)
		}

		return false, nil
	}

	return true, nil
}

func (th *transformerHandler) setupExecution(ctx context.Context) error {
	if th.setupComplete {
		return nil
	}

	if th.nativeFunction == nil {
		jsContext, cb, err := ucv8go.NewJSContext(ctx, th.authzClient, auditlog.TransformerCustom, newPolicySecretResolver(th.s))
		if err != nil {
			return ucerr.Wrap(err)
		}
		th.jsContext = jsContext
		th.jsScript = ""
		th.consoleBuilder = cb
	}
	th.setupComplete = true

	return nil
}

func (th *transformerHandler) transform(ctx context.Context, index int) (string, error) {
	if th.nativeFunction != nil {
		return (*th.nativeFunction)(th.data[index], th.transformer.Parameters), nil
	} else if th.jsContext == nil {
		return "", ucerr.Errorf("transformer called with nil jsContext for non-native transformer %v", th.transformer.ID)
	}

	// encode the arguments and format the script to run
	encodedData, err := th.encodeData(th.data[index])
	if err != nil {
		return "", ucerr.Wrap(err)
	}

	if th.jsScript == "" {
		// first time through, we also load the transformer functions
		th.jsScript = fmt.Sprintf("%s;\ntransform(%v, %v)", th.transformer.Function, encodedData, th.transformer.Parameters)
	} else {
		th.jsScript = fmt.Sprintf("transform(%v, %v)", encodedData, th.transformer.Parameters)
	}

	// run the script
	jsResult, err := th.jsContext.RunScript(th.jsScript, fmt.Sprintf("%v.js", th.transformer.ID))
	if err != nil {
		uclog.Warningf(ctx, "failed to execute transformer %v (%s): %v", th.transformer.ID, th.jsScript, err)
		logTransformerError(ctx, th.transformer.ID, th.transformer.Version)
		wp := jsonclient.Error{
			StatusCode: http.StatusBadRequest,
			Body:       fmt.Sprintf("error executing transformer: %v", err),
		}
		return "", ucerr.Wrap(wp)

	}

	// return a string representation
	if jsResult.IsArray() || jsResult.IsObject() {
		marshaledResult, err := jsResult.MarshalJSON()
		if err != nil {
			return "", ucerr.Wrap(err)
		}
		return string(marshaledResult), nil
	}

	return jsResult.String(), nil
}
