package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/uclog"
)

func main() {
	ctx := context.Background()

	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelDebug, uclog.LogLevelVerbose, "azcli")
	defer logtransports.Close()

	if len(os.Args) < 3 {
		uclog.Errorf(ctx, "Usage: azcli <tenant URL> <client ID> <client secret> <command> [args]")
		uclog.Debugf(ctx, "  listobjtypes - list object types")
		uclog.Debugf(ctx, "  createobjecttype <name> - create an object type")
		uclog.Debugf(ctx, "")
		uclog.Debugf(ctx, "  listedgetypes - list edge types")
		uclog.Debugf(ctx, "  createedgetype <name> <source type ID> <target type ID> <org ID> <attributes> - create an edge type")
		uclog.Debugf(ctx, "    attributes: comma-delimited list of <name>:<d|i|p> where d=direct, i=inherit, p=propagate")
		uclog.Debugf(ctx, "")
		uclog.Debugf(ctx, "  listedges <obj ID> - list edges to/from a specific object")
		uclog.Debugf(ctx, "  createedge <type ID> <source ID> <target ID> <org ID> - create an edge")
		uclog.Debugf(ctx, "  deleteedge <edge ID> - delete an edge")
		uclog.Debugf(ctx, "")
		uclog.Debugf(ctx, "  listobjs - list objects")
		uclog.Debugf(ctx, "  createobj <type ID> <alias> <org ID> - create an object")
		uclog.Debugf(ctx, "  deleteobj <obj ID> - delete an object")
		uclog.Debugf(ctx, "")
		uclog.Debugf(ctx, "  listorgs - list organizations")
		uclog.Debugf(ctx, "  createorg <name> <region> - create an organization")
		uclog.Debugf(ctx, "")
		uclog.Debugf(ctx, "  checkattr <src obj ID> <target obj ID> <attribute name> - check for an attribute path between two objects")
		uclog.Debugf(ctx, "  listattr <src obj ID> <target obj ID> - list attributes between two objects")
		uclog.Debugf(ctx, "")
		uclog.Debugf(ctx, "  migrateobj <obj ID> <org ID> - migrate a user to an organization")
		uclog.Debugf(ctx, "  migrateedgetype <et ID> <org ID> - migrate an edge type to an organization")
		uclog.Debugf(ctx, "")

		// TODO: this is a weird place for this but didn't seem worth building a plexclient binary yet
		uclog.Debugf(ctx, "  sendinvite <email to send to> <IDP user ID sending invite> <IDP sender name> <IDP sender email> <IDP user ID to delegate>")
		return
	}

	tenantURL := os.Args[1]
	clientID := os.Args[2]
	clientSecret := os.Args[3]
	command := os.Args[4]

	// this doesn't require an actual authz client :)
	if command == "sendinvite" {
		sendInvite(ctx)
		return
	}

	tokenSource, err := jsonclient.ClientCredentialsForURL(tenantURL, clientID, clientSecret, nil)
	if err != nil {
		uclog.Fatalf(ctx, "failed to create token source: %v", err)
	}
	azc, err := authz.NewClient(tenantURL, authz.JSONClient(tokenSource))
	if err != nil {
		uclog.Fatalf(ctx, "failed to create authz client: %v", err)
	}

	switch command {

	case "listobjtypes":
		ots, err := azc.ListObjectTypes(ctx)
		if err != nil {
			uclog.Fatalf(ctx, "Failed to list object types: %v", err)
		}
		for _, ot := range ots {
			uclog.Infof(ctx, "%+v", ot)
		}

	case "createobjecttype":
		name := os.Args[5]
		ot, err := azc.CreateObjectType(ctx, uuid.Must(uuid.NewV4()), name)
		if err != nil {
			uclog.Fatalf(ctx, "failed to create object type: %v", err)
		}
		uclog.Infof(ctx, "created %+v", ot)

	case "listedgetypes":
		ets, err := azc.ListEdgeTypes(ctx)
		if err != nil {
			uclog.Fatalf(ctx, "Failed to list edge types: %v", err)
		}
		for _, et := range ets {
			uclog.Infof(ctx, "%+v", et)
		}

	case "createedgetype":
		name := os.Args[5]
		srcTypeID := uuid.Must(uuid.FromString(os.Args[6]))
		tgtTypeID := uuid.Must(uuid.FromString(os.Args[7]))
		organizationID, err := uuid.FromString(os.Args[8])
		if err != nil {
			organizationID = uuid.Nil
		}

		var attrs []authz.Attribute
		if len(os.Args) > 9 {
			attr := os.Args[9]

			for a := range strings.SplitSeq(attr, ",") {
				if a == "" {
					continue
				}
				parts := strings.Split(a, ":")
				if len(parts) != 2 {
					uclog.Fatalf(ctx, "invalid attribute: %s", a)
				}
				at := authz.Attribute{Name: parts[0]}
				switch parts[1] {
				case "d":
					at.Direct = true
				case "i":
					at.Inherit = true
				case "p":
					at.Propagate = true
				}

				attrs = append(attrs, at)
			}
		}

		et, err := azc.CreateEdgeType(ctx, uuid.Must(uuid.NewV4()), srcTypeID, tgtTypeID, name, attrs, authz.OrganizationID(organizationID))
		if err != nil {
			uclog.Fatalf(ctx, "failed to create edge type: %v", err)
		}
		uclog.Infof(ctx, "created %+v", et)

	case "listobjs":
		// TODO: add native pagination support to CLI?
		cursor := pagination.CursorBegin
		for {
			resp, err := azc.ListObjects(ctx, authz.Pagination(pagination.StartingAfter(cursor)))
			if err != nil {
				uclog.Fatalf(ctx, "failed to list objects: %v", err)
			}
			for _, obj := range resp.Data {
				uclog.Infof(ctx, "%+v", obj)
			}
			if !resp.HasNext {
				break
			}
			cursor = resp.Next
		}

	case "listedges":
		objID := uuid.Must(uuid.FromString(os.Args[5]))
		cursor := pagination.CursorBegin
		for {
			resp, err := azc.ListEdgesOnObject(ctx, objID, authz.Pagination(pagination.StartingAfter(cursor)))
			if err != nil {
				uclog.Fatalf(ctx, "failed to list edges: %v", err)
			}
			for _, e := range resp.Data {
				uclog.Infof(ctx, "%+v", e)
			}
			if !resp.HasNext {
				break
			}
			cursor = resp.Next
		}

	case "createedge":
		typID := uuid.Must(uuid.FromString(os.Args[5]))
		srcID := uuid.Must(uuid.FromString(os.Args[6]))
		tgtID := uuid.Must(uuid.FromString(os.Args[7]))

		edge, err := azc.CreateEdge(ctx, uuid.Must(uuid.NewV4()), srcID, tgtID, typID)
		if err != nil {
			uclog.Fatalf(ctx, "failed to create edge: %v", err)
		}
		uclog.Infof(ctx, "%+v", edge)

	case "deleteedge":
		edgeID := uuid.Must(uuid.FromString(os.Args[5]))
		if err := azc.DeleteEdge(ctx, edgeID); err != nil {
			uclog.Fatalf(ctx, "failed to delete edge: %v", err)
		}
		uclog.Infof(ctx, "deleted %v", edgeID)

	case "createobj":
		typeID := uuid.Must(uuid.FromString(os.Args[5]))
		alias := os.Args[6]
		organizationID, err := uuid.FromString(os.Args[7])
		if err != nil {
			organizationID = uuid.Nil
		}

		obj, err := azc.CreateObject(ctx, uuid.Must(uuid.NewV4()), typeID, alias, authz.OrganizationID(organizationID))
		if err != nil {
			uclog.Fatalf(ctx, "failed to create object: %v", err)
		}
		uclog.Infof(ctx, "created %+v", obj)

	case "deleteobj":
		objID := uuid.Must(uuid.FromString(os.Args[5]))
		if err := azc.DeleteObject(ctx, objID); err != nil {
			uclog.Fatalf(ctx, "failed to delete object: %v", err)
		}
		uclog.Infof(ctx, "deleted %v", objID)

	case "createorg":
		name := os.Args[5]
		reg := region.DataRegion(os.Args[6])
		org, err := azc.CreateOrganization(ctx, uuid.Must(uuid.NewV4()), name, reg)
		if err != nil {
			uclog.Fatalf(ctx, "Failed to create organization: %v", err)
		}
		uclog.Infof(ctx, "created %+v", org)

	case "listorgs":
		resp, err := azc.ListOrganizations(ctx)
		if err != nil {
			uclog.Fatalf(ctx, "Failed to list organizations: %v", err)
		}
		for _, org := range resp {
			uclog.Infof(ctx, "%+v", org)
		}

	case "migrateobj":
		objID := uuid.Must(uuid.FromString(os.Args[5]))
		orgID := uuid.Must(uuid.FromString(os.Args[6]))
		obj, err := azc.AddOrganizationToObject(ctx, objID, orgID)
		if err != nil {
			uclog.Fatalf(ctx, "failed to migrate object: %v", err)
		}
		uclog.Infof(ctx, "migrated %+v", obj)

	case "migrateedgetype":
		etID := uuid.Must(uuid.FromString(os.Args[5]))
		orgID := uuid.Must(uuid.FromString(os.Args[6]))
		et, err := azc.AddOrganizationToEdgeType(ctx, etID, orgID)
		if err != nil {
			uclog.Fatalf(ctx, "failed to migrate edge type: %v", err)
		}
		uclog.Infof(ctx, "migrated %+v", et)

	case "checkattr":
		srcObjID := uuid.Must(uuid.FromString(os.Args[5]))
		tgtObjID := uuid.Must(uuid.FromString(os.Args[6]))
		attr := os.Args[7]
		res, err := azc.CheckAttribute(ctx, srcObjID, tgtObjID, attr)
		if err != nil {
			uclog.Fatalf(ctx, "failed to check attribute: %v", err)
		}
		uclog.Infof(ctx, "result: %v", res)

	case "listattr":
		srcObjID := uuid.Must(uuid.FromString(os.Args[5]))
		tgtObjID := uuid.Must(uuid.FromString(os.Args[6]))
		res, err := azc.ListAttributes(ctx, srcObjID, tgtObjID)
		if err != nil {
			uclog.Fatalf(ctx, "failed to list attributes: %v", err)
		}
		uclog.Infof(ctx, "result: %v", res)
	default:
		uclog.Fatalf(ctx, "unknown command: %s", command)
	}

}

func sendInvite(ctx context.Context) {
	tenantURL := os.Args[1]
	clientID := os.Args[2]
	clientSecret := os.Args[3]
	// command is 4

	emailTo := os.Args[5]
	fromUserID := os.Args[6]
	fromName := os.Args[7]
	fromEmail := os.Args[8]
	forUserID := os.Args[9]

	ts, err := jsonclient.ClientCredentialsForURL(tenantURL, clientID, clientSecret, nil)
	if err != nil {
		uclog.Fatalf(ctx, "failed to create token source: %v", err)
	}
	client := jsonclient.New(tenantURL, ts)

	q := url.Values{}
	q.Add("client_id", clientID)
	q.Add("to", emailTo)
	q.Add("from", fromUserID)
	q.Add("from_name", fromName)
	q.Add("from_email", fromEmail)
	q.Add("for", forUserID)

	type emptyResponse struct{}
	var resp emptyResponse
	if err := client.Get(ctx, fmt.Sprintf("/delegation/invite?%s", q.Encode()), &resp); err != nil {
		uclog.Fatalf(ctx, "failed to send invite: %v", err)
	}

	uclog.Infof(ctx, "invite sent")
}
