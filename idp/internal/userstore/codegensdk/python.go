package codegensdk

import (
	"context"
	"fmt"

	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/ucerr"
)

type pythonSDKGenerator struct {
	baseSDKGenerator
}

func newPythonSDKGenerator(ctx context.Context, s *storage.Storage) sdkGenerator {
	return pythonSDKGenerator{
		baseSDKGenerator: newBaseSDKGenerator(ctx, s),
	}
}

func (pythonSDKGenerator) getFormattedSource(b []byte) ([]byte, error) {
	return b, nil
}

func (pythonSDKGenerator) getFuncArgComponent(
	funcArg string,
	isArray bool,
) string {
	if isArray {
		return fmt.Sprintf("%s: list[str]", funcArg)
	}
	return fmt.Sprintf("%s: str", funcArg)
}

func (pythonSDKGenerator) getTemplate() string {
	return `# *** WARNING! ***
# This file is auto-generated and will be overwritten when a schema is modified.
# DO NOT EDIT.
#
# Origin: << .Origin >>
# Generated at: << .GeneratedAt >>
from __future__ import annotations

import datetime
import enum
import json
import uuid
from dataclasses import dataclass, asdict

from usercloudssdk.client import Client
from usercloudssdk.asyncclient import AsyncClient

# composite value types

<<- range .TemplateDataTypes >>

@dataclass
class << .Name >>:
<<- range .Fields >>
	<< .Name >>: << .TypeName >> | None = None
<<- end >>

	@classmethod
	def from_json(cls, json_data: dict) -> << .Name >>:
		return cls(
<<- range .Fields >>
			<< .Name >>=json_data.get("<< .Name >>"),
<<- end >>
		)
<<- end >>

# accessor types

<<- range .TemplateAccessors >>

@dataclass
class << .ObjectName >>:
<<- range .ObjectMembers >>
	<<- if .IsArray >>
	<< .Name >>: list[<< .TypeName >>] | None = None
	<<- else if eq .TypeName "str" >>
	<< .Name >>: str = ""
	<<- else >>
	<< .Name >>: << .TypeName >> | None = None
	<<- end >>
<<- end >>

	def to_dict(self) -> dict:
		return asdict(self)

class << .ObjectName >>Response:
	data: list[<< .ObjectName >>]
	has_next: bool
	has_prev: bool
	next: str | None
	prev: str | None

	def __init__(
		self,
		data: list[<< .ObjectName >>],
		has_next: bool,
		has_prev: bool,
		next: str | None = None,
		prev: str | None = None,
	) -> None:
		self.data = data
		self.has_next = has_next
		self.has_prev = has_prev
		self.next = next
		self.prev = prev

	def to_dict(self) -> dict:
		return asdict(self)
<<- end >>

# mutator types

<<- range .TemplateMutators >>

@dataclass
class << .ObjectName >>:
<<- range .ObjectMembers >>
	<<- if .IsArray >>
	<< .Name >>: list[<< .TypeName >>] | None = None
	<<- else if eq .TypeName "str" >>
	<< .Name >>: str = ""
	<<- else >>
	<< .Name >>: << .TypeName >> | None = None
	<<- end >>
<<- end >>

	def to_dict(self) -> dict:
		return asdict(self)
<<- end >>

# accessors

<<- range .TemplateAccessors >>

# << .FunctionName >>
# selector: "<< .WhereClause >>"

def << .FunctionName >>(self, << .FunctionArgumentString >>limit: int = 0, starting_after: str | None = None, ending_before: str | None = None, sort_key: str | None = None, sort_order: str | None = None) -> << .ObjectName >>Response:
	resp = self.ExecuteAccessor(
		accessor_id="<< .AccessorID >>",
		context={},
		selector_values=[<< .SelectorArgumentString >>],
		limit=limit,
		starting_after=starting_after,
		ending_before=ending_before,
		sort_key=sort_key,
		sort_order=sort_order,
	)

	ret = []
	for resp_data in resp["data"]:
		json_data = json.loads(resp_data)
		obj = << .ObjectName >>()
		<<- range .ObjectMembers >>
		if json_data.get("<< .Name >>") is not None:
			<<- if .IsArray >>
				<<- if eq .TypeName "str" >>
			obj.<< .Name >> = [x for x in json.loads(json_data["<< .Name >>"])]
				<<- else if eq .TypeName "bool" >>
			obj.<< .Name >> = [x == "true" for x in json.loads(json_data["<< .Name >>"])]
				<<- else if eq .TypeName "int" >>
			obj.<< .Name >> = [int(x) for x in json.loads(json_data["<< .Name >>"])]
				<<- else if eq .TypeName "datetime.datetime" >>
			obj.<< .Name >> = [datetime.datetime.strptime(x, "%Y-%m-%dT%H:%M:%SZ") for x in json.loads(json_data["<< .Name >>"])]
				<<- else if eq .TypeName "uuid.UUID" >>
			obj.<< .Name >> = [uuid.UUID(x) for x in json.loads(json_data["<< .Name >>"])]
				<<- else >>
			obj.<< .Name >> = [<< .TypeName >>.from_json(x) for x in json.loads(json_data["<< .Name >>"])]
				<<- end >>
			<<- else if eq .TypeName "str" >>
			obj.<< .Name >> = json_data["<< .Name >>"]
			<<- else if eq .TypeName "bool" >>
			obj.<< .Name >> = json_data["<< .Name >>"] == "true"
			<<- else if eq .TypeName "int" >>
			obj.<< .Name >> = int(json_data["<< .Name >>"])
			<<- else if eq .TypeName "datetime.datetime" >>
			obj.<< .Name >> = datetime.datetime.strptime(json_data["<< .Name >>"], "%Y-%m-%dT%H:%M:%SZ")
			<<- else if eq .TypeName "uuid.UUID" >>
			obj.<< .Name >> = uuid.UUID(json_data["<< .Name >>"])
			<<- else >>
			obj.<< .Name >> = << .TypeName >>.from_json(json_data["<< .Name >>"])
			<<- end >>
		<<- end >>

		ret.append(obj)

	accessor_resp = << .ObjectName >>Response(
		data=ret,
		has_next=resp["has_next"],
		has_prev=resp["has_prev"],
		next=resp.get("next"),
		prev=resp.get("prev"),
	)

	return accessor_resp

Client.<< .FunctionName >> = << .FunctionName >>

# << .FunctionName >>Async
# selector: "<< .WhereClause >>"

async def << .FunctionName >>Async(self, << .FunctionArgumentString >>limit: int = 0, starting_after: str | None = None, ending_before: str | None = None, sort_key: str | None = None, sort_order: str | None = None) -> << .ObjectName >>Response:
	resp = await self.ExecuteAccessorAsync(
		accessor_id="<< .AccessorID >>",
		context={},
		selector_values=[<< .SelectorArgumentString >>],
		limit=limit,
		starting_after=starting_after,
		ending_before=ending_before,
		sort_key=sort_key,
		sort_order=sort_order,
	)

	ret = []
	for resp_data in resp["data"]:
		json_data = json.loads(resp_data)
		obj = << .ObjectName >>()
		<<- range .ObjectMembers >>
		if json_data.get("<< .Name >>") is not None:
			<<- if .IsArray >>
				<<- if eq .TypeName "str" >>
			obj.<< .Name >> = [x for x in json.loads(json_data["<< .Name >>"])]
				<<- else if eq .TypeName "bool" >>
			obj.<< .Name >> = [x == "true" for x in json.loads(json_data["<< .Name >>"])]
				<<- else if eq .TypeName "int" >>
			obj.<< .Name >> = [int(x) for x in json.loads(json_data["<< .Name >>"])]
				<<- else if eq .TypeName "datetime.datetime" >>
			obj.<< .Name >> = [datetime.datetime.strptime(x, "%Y-%m-%dT%H:%M:%SZ") for x in json.loads(json_data["<< .Name >>"])]
				<<- else if eq .TypeName "uuid.UUID" >>
			obj.<< .Name >> = [uuid.UUID(x) for x in json.loads(json_data["<< .Name >>"])]
				<<- else >>
			obj.<< .Name >> = [<< .TypeName >>.from_json(x) for x in json.loads(json_data["<< .Name >>"])]
				<<- end >>
			<<- else if eq .TypeName "str" >>
			obj.<< .Name >> = json_data["<< .Name >>"]
			<<- else if eq .TypeName "bool" >>
			obj.<< .Name >> = json_data["<< .Name >>"] == "true"
			<<- else if eq .TypeName "int" >>
			obj.<< .Name >> = int(json_data["<< .Name >>"])
			<<- else if eq .TypeName "datetime.datetime" >>
			obj.<< .Name >> = datetime.datetime.strptime(json_data["<< .Name >>"], "%Y-%m-%dT%H:%M:%SZ")
			<<- else if eq .TypeName "uuid.UUID" >>
			obj.<< .Name >> = uuid.UUID(json_data["<< .Name >>"])
			<<- else >>
			obj.<< .Name >> = << .TypeName >>.from_json(json_data["<< .Name >>"])
			<<- end >>
		<<- end >>

		ret.append(obj)

	accessor_resp = << .ObjectName >>Response(
		data=ret,
		has_next=resp["has_next"],
		has_prev=resp["has_prev"],
		next=resp.get("next"),
		prev=resp.get("prev"),
	)

	return accessor_resp

AsyncClient.<< .FunctionName >>Async = << .FunctionName >>Async
<<- end >>

# mutators

<<- $mutators := .TemplateMutators >>
<<- $purposes := .Purposes >>

<<- range $mutator := $mutators >>
<<- range $purpose := $purposes >>

# << $mutator.FunctionName >>For<< $purpose.PascalCaseName >>Purpose
# selector: "<< $mutator.WhereClause >>"

def << $mutator.FunctionName >>For<< $purpose.PascalCaseName >>Purpose(self, obj: << $mutator.ObjectName >>, << $mutator.FunctionArgumentString >>) -> list:
	row_data = {}
	<<- range $mutator.ObjectMembers >>
	row_data["<< .Name >>"] = {
		"<< .MutatorValueField >>": obj.<< .Name >>,
		"purpose_additions": [
			{
				"name": "<< $purpose.Name >>"
			}
		]
	}
	<<- end >>

	resp = self.ExecuteMutator(
		"<< $mutator.MutatorID >>",
		{},
		[<< $mutator.SelectorArgumentString >>],
		row_data
	)
	return resp

Client.<< $mutator.FunctionName >>For<< $purpose.PascalCaseName >>Purpose = << $mutator.FunctionName >>For<< $purpose.PascalCaseName >>Purpose

# << $mutator.FunctionName >>For<< $purpose.PascalCaseName >>PurposeAsync
# selector: "<< $mutator.WhereClause >>"

async def << $mutator.FunctionName >>For<< $purpose.PascalCaseName >>PurposeAsync(self, obj: << $mutator.ObjectName >>, << $mutator.FunctionArgumentString >>) -> list:
	row_data = {}
	<<- range $mutator.ObjectMembers >>
	row_data["<< .Name >>"] = {
		"<< .MutatorValueField >>": obj.<< .Name >>,
		"purpose_additions": [
			{
				"name": "<< $purpose.Name >>"
			}
		]
	}
	<<- end >>

	resp = await self.ExecuteMutatorAsync(
		"<< $mutator.MutatorID >>",
		{},
		[<< $mutator.SelectorArgumentString >>],
		row_data
	)
	return resp

AsyncClient.<< $mutator.FunctionName >>For<< $purpose.PascalCaseName >>PurposeAsync = << $mutator.FunctionName >>For<< $purpose.PascalCaseName >>PurposeAsync
<<- end >>
<<- end >>

class Purpose(enum.Enum):
<<- range .Purposes >>
	<< .AllCapsName >> = "<< .Name >>"
<<- end >>
<< range .TemplateMutators >>
# << .FunctionName >>ForPurposes
# selector: "<< .WhereClause >>"

def << .FunctionName >>ForPurposes(self, purposes: list[Purpose], obj: << .ObjectName >>, << .FunctionArgumentString >>) -> list:
	row_data = {}
	purpose_additions = list(map(lambda p: {"name": p.value}, purposes))
	<<- range .ObjectMembers >>
	row_data["<< .Name >>"] = {
		"<< .MutatorValueField >>": obj.<< .Name >>,
		"purpose_additions": purpose_additions
	}
	<<- end >>

	resp = self.ExecuteMutator(
		"<< .MutatorID >>",
		{},
		[<< .SelectorArgumentString >>],
		row_data
	)
	return resp

Client.<< .FunctionName >>ForPurposes = << .FunctionName >>ForPurposes

# << .FunctionName >>ForPurposesAsync
# selector: "<< .WhereClause >>"

async def << .FunctionName >>ForPurposesAsync(self, purposes: list[Purpose], obj: << .ObjectName >>, << .FunctionArgumentString >>) -> list:
	row_data = {}
	<<- range .ObjectMembers >>
	row_data["<< .Name >>"] = {
		"<< .MutatorValueField >>": obj.<< .Name >>,
		"purpose_additions": list(map(lambda p: {"name": p.value}, purposes))
	}
	<<- end >>

	resp = await self.ExecuteMutatorAsync(
		"<< .MutatorID >>",
		{},
		[<< .SelectorArgumentString >>],
		row_data
	)
	return resp

AsyncClient.<< .FunctionName >>ForPurposesAsync = << .FunctionName >>ForPurposesAsync
<< end >>

<<- if .IncludeExample >>
# example

def example():
	# Init client from env variables: TENANT_URL, CLIENT_ID & CLIENT_SECRET
	client = Client.from_env()

	uid = client.CreateUser()
	print(f"Created user with id: {uid}")

	users = client.GetUsers([uid])
	user = users[0]
	print(f"User: {user}")

	user.name = "New Name"
	client.UpdateUserForOperationalPurpose(user, uid)
	users = client.GetUsers([uid])
	user = users[0]
	print(f"User: {user}")
<<- end >>
`
}

func (pythonSDKGenerator) setTypeName(omd objectMemberData) objectMemberData {
	switch omd.DataType.ConcreteDataTypeID {
	case datatype.String.ID:
		omd.TypeName = "str"
	case datatype.Boolean.ID:
		omd.TypeName = "bool"
	case datatype.Integer.ID:
		omd.TypeName = "int"
	case datatype.Date.ID, datatype.Timestamp.ID:
		omd.TypeName = "datetime.datetime"
	case datatype.UUID.ID:
		omd.TypeName = "uuid.UUID"
	case datatype.Composite.ID:
		// the passed in type name is used
	}

	return omd
}

// CodegenPythonSDK generates a Python SDK for the given IDP client
func CodegenPythonSDK(ctx context.Context, s *storage.Storage, includeExample bool) ([]byte, error) {
	sdkGenerator := newPythonSDKGenerator(ctx, s)
	output, err := generateSDK(sdkGenerator, includeExample)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return output, nil
}
