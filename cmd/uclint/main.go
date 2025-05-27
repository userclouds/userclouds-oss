package main

import (
	"golang.org/x/tools/go/analysis/multichecker"

	"userclouds.com/tools/analyzer"
)

func main() {
	multichecker.Main(
		analyzer.BaseModelAnalyzer,
		analyzer.CompressAnalyzer,
		analyzer.DBTagAnalyzer,
		analyzer.ErrorsAsAnalyzer,
		analyzer.HandlerAnalyzer,
		analyzer.HTTPErrorAnalyzer,
		analyzer.ImportAnalyzer,
		analyzer.JSONTagAnalyzer,
		analyzer.MakeMapAnalyzer,
		analyzer.ReturnUCErrAnalyzer,
		analyzer.SQLAnalyzer,
		analyzer.UCWrapperAnalyzer,
		analyzer.UTCAnalyzer,
		analyzer.ValidateAnalyzer,
		analyzer.YAMLTagAnalyzer,
		analyzer.PaginationAnalyzer,
		analyzer.PaginationResultAnalyzer,
		analyzer.FriendlyErrorAnalyzer,
		analyzer.ErrorFormatAnalyzer,
		analyzer.NilAppendAnalyzer,
		analyzer.ShadowImportAnalyzer,
	)
}
