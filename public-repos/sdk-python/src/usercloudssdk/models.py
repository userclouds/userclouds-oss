from __future__ import annotations

import datetime
import uuid
from dataclasses import asdict, dataclass

import iso8601

from . import ucjson
from .constants import (
    ColumnIndexType,
    DataLifeCycleState,
    DataType,
    PolicyType,
    TransformType,
)


class User:
    id: uuid.UUID
    profile: dict

    def __init__(self, id: uuid.UUID, profile: dict) -> None:
        self.id = id
        self.profile = profile

    def to_json(self) -> str:
        return ucjson.dumps(
            {
                "id": str(self.id),
                "profile": self.profile,
            }
        )

    def __str__(self) -> str:
        return f"User {self.id} - {self.profile}"

    def __repr__(self) -> str:
        return f"User(id={self.id}, profile={self.profile})"

    @classmethod
    def from_json(cls, json_data: dict) -> User:
        return cls(
            id=uuid.UUID(json_data["id"]),
            profile=json_data["profile"],
        )


class UserResponse:
    id: uuid.UUID
    updated_at: datetime.datetime
    profile: dict
    organization_id: uuid.UUID

    def __init__(
        self,
        id: uuid.UUID,
        updated_at: datetime.datetime,
        profile: dict,
        organization_id: uuid.UUID,
    ) -> None:
        self.id = id
        self.updated_at = updated_at
        self.profile = profile
        self.organization_id = organization_id

    def to_json(self) -> str:
        return ucjson.dumps(
            {
                "id": str(self.id),
                "updated_at": self.updated_at.isoformat(),
                "profile": self.profile,
                "organization_id": str(self.organization_id),
            }
        )

    @classmethod
    def from_json(cls, json_data: dict) -> UserResponse:
        return cls(
            id=uuid.UUID(json_data["id"]),
            updated_at=datetime.datetime.fromtimestamp(json_data["updated_at"]),
            profile=json_data["profile"],
            organization_id=uuid.UUID(json_data["organization_id"]),
        )


class UserSelectorConfig:
    where_clause: str

    def __init__(self, where_clause: str) -> None:
        self.where_clause = where_clause

    def __repr__(self) -> str:
        return f"UserSelectorConfig({self.where_clause})"

    def __str__(self) -> str:
        return f"UserSelectorConfig: {self.where_clause}"

    def to_json(self) -> str:
        return ucjson.dumps({"where_clause": self.where_clause})

    @classmethod
    def from_json(cls, json_data: dict) -> UserSelectorConfig:
        return cls(where_clause=json_data["where_clause"])


class ResourceID:
    def __init__(self, id="", name="") -> None:
        if id:
            setattr(self, "id", id)
        if name:
            setattr(self, "name", name)

    def __repr__(self) -> str:
        if hasattr(self, "id") and hasattr(self, "name"):
            return f"ResourceID(id={self.id}, name={self.name})"
        if hasattr(self, "id"):
            return f"ResourceID({self.id})"
        if hasattr(self, "name"):
            return f"ResourceID({self.name})"
        return "ResourceID()"

    def isValid(self) -> bool:
        return hasattr(self, "id") or hasattr(self, "name")

    @classmethod
    def from_json(cls, json_data: dict) -> ResourceID:
        return cls(id=json_data["id"], name=json_data["name"])

    def to_dict(self) -> dict:
        rsc_id = {}
        if hasattr(self, "id"):
            rsc_id["id"] = self.id
        if hasattr(self, "name"):
            rsc_id["name"] = self.name
        if not rsc_id:
            raise ValueError("ResourceID is empty")
        return rsc_id


class CompositeField:
    data_type: ResourceID
    name: str
    camel_case_name: str
    struct_name: str
    required: bool
    ignore_for_uniqueness: bool

    def __init__(
        self,
        data_type: ResourceID,
        name: str,
        camel_case_name: str = "",
        struct_name: str = "",
        required: bool = False,
        ignore_for_uniqueness: bool = False,
    ) -> None:
        self.data_type = data_type
        self.name = name
        self.camel_case_name = camel_case_name
        self.struct_name = struct_name
        self.required = required
        self.ignore_for_uniqueness = ignore_for_uniqueness

    def __str__(self) -> str:
        return f"CompositeField: {self.name} [{self.data_type}]"

    def __repr__(self) -> str:
        return f"CompositeField(data_type={self.data_type}, name={self.name}, camel_case_name={self.camel_case_name}, struct_name={self.struct_name}, required={self.required}, ignore_for_uniqueness={self.ignore_for_uniqueness})"

    def to_json(self) -> str:
        return ucjson.dumps(
            {
                "data_type": self.data_type,
                "name": self.name,
                "camel_case_name": self.camel_case_name,
                "struct_name": self.struct_name,
                "required": self.required,
                "ignore_for_uniqueness": self.ignore_for_uniqueness,
            }
        )

    @classmethod
    def from_json(cls, json_data: dict) -> ColumnField:
        return cls(
            data_type=ResourceID.from_json(json_data["data_type"]),
            name=json_data["name"],
            camel_case_name=json_data["camel_case_name"],
            struct_name=json_data["struct_name"],
            required=json_data["required"],
            ignore_for_uniqueness=json_data["ignore_for_uniqueness"],
        )


class CompositeAttributes:
    include_id: bool
    fields: list[CompositeField]

    def __init__(
        self,
        include_id: bool,
        fields: list[CompositeField],
    ) -> None:
        self.include_id = include_id
        self.fields = fields

    def __str__(self) -> str:
        return (
            f"CompositeAttributes: include_id={self.include_id}, fields={self.fields}"
        )

    def __repr__(self) -> str:
        return f"CompsositeAttributes(include_id={self.include_id}, fields={self.fields!r})"

    def to_json(self) -> str:
        return ucjson.dumps(
            {
                "include_id": self.include_id,
                "fields": [field.to_json() for field in self.fields],
            }
        )

    @classmethod
    def from_json(cls, json_data: dict) -> CompositeAttributes:
        return cls(
            include_id=json_data["include_id"],
            fields=json_data["fields"],
        )


class ColumnDataType:
    id: uuid.UUID
    name: str
    description: str
    is_composite_field_type: bool
    is_native: bool
    composite_attributes: CompositeAttributes

    def __init__(
        self,
        id: uuid.UUID,
        name: str,
        description: str,
        is_composite_field_type: bool = False,
        is_native: bool = False,
        composite_attributes: CompositeAttributes | None = None,
    ) -> None:
        self.id = id
        self.name = name
        self.description = description
        self.is_composite_field_type = is_composite_field_type
        self.is_native = is_native
        if composite_attributes is None:
            self.composite_attributes = CompositeAttributes(
                include_id=False,
                fields=[],
            )
        else:
            self.composite_attributes = composite_attributes

    def to_json(self) -> str:
        return ucjson.dumps(
            {
                "id": str(self.id),
                "name": self.name,
                "description": self.description,
                "is_composite_field_type": self.is_composite_field_type,
                "is_native": self.is_native,
                "composite_attributes": self.composite_attributes,
            }
        )

    @classmethod
    def from_json(cls, json_data: dict) -> ColumnDataType:
        return cls(
            id=uuid.UUID(json_data["id"]),
            name=json_data["name"],
            description=json_data["description"],
            is_composite_field_type=json_data["is_composite_field_type"],
            is_native=json_data["is_native"],
            composite_attributes=json_data["composite_attributes"],
        )

    def __str__(self) -> str:
        return f"ColumnDataType {self.name} [{self.description}] - {self.id}"

    def __repr__(self) -> str:
        return f"ColumnDataType(id={self.id}, name={self.name}, description={self.description})"


class ColumnField:
    type: DataType
    name: str
    camel_case_name: str
    struct_name: str
    required: bool
    ignore_for_uniqueness: bool

    def __init__(
        self,
        type: str | DataType,
        name: str,
        camel_case_name: str = "",
        struct_name: str = "",
        required: bool = False,
        ignore_for_uniqueness: bool = False,
    ) -> None:
        self.type = DataType(type)
        self.name = name
        self.camel_case_name = camel_case_name
        self.struct_name = struct_name
        self.required = required
        self.ignore_for_uniqueness = ignore_for_uniqueness

    def __str__(self) -> str:
        return f"ColumnField: {self.name} [{self.type}]"

    def __repr__(self) -> str:
        return f"ColumnField(type={self.type}, name={self.name}, camel_case_name={self.camel_case_name}, struct_name={self.struct_name}, required={self.required}, ignore_for_uniqueness={self.ignore_for_uniqueness})"

    def to_json(self) -> str:
        return ucjson.dumps(
            {
                "type": self.type.value,
                "name": self.name,
                "camel_case_name": self.camel_case_name,
                "struct_name": self.struct_name,
                "required": self.required,
                "ignore_for_uniqueness": self.ignore_for_uniqueness,
            }
        )

    @classmethod
    def from_json(cls, json_data: dict) -> ColumnField:
        return cls(
            type=DataType(json_data["type"]),
            name=json_data["name"],
            camel_case_name=json_data["camel_case_name"],
            struct_name=json_data["struct_name"],
            required=json_data["required"],
            ignore_for_uniqueness=json_data["ignore_for_uniqueness"],
        )


class ColumnConstraints:
    immutable_required: bool
    partial_updates: bool
    unique_id_required: bool
    unique_required: bool
    fields: list[ColumnField]

    def __init__(
        self,
        immutable_required: bool,
        partial_updates: bool,
        unique_id_required: bool,
        unique_required: bool,
        fields: list[ColumnField],
    ) -> None:
        self.immutable_required = immutable_required
        self.partial_updates = partial_updates
        self.unique_id_required = unique_id_required
        self.unique_required = unique_required
        self.fields = fields

    def __str__(self) -> str:
        return f"ColumnConstraints: immutable_required={self.immutable_required}, partial_updates={self.partial_updates}, unique_id_required={self.unique_id_required}, unique_required={self.unique_required}, fields={self.fields}"

    def __repr__(self) -> str:
        return f"ColumnConstraints(immutable_required={self.immutable_required}, partial_updates={self.partial_updates}, unique_id_required={self.unique_id_required}, unique_required={self.unique_required}, fields={self.fields!r})"

    def to_json(self) -> str:
        return ucjson.dumps(
            {
                "immutable_required": self.immutable_required,
                "partial_updates": self.partial_updates,
                "unique_id_required": self.unique_id_required,
                "unique_required": self.unique_required,
                "fields": [field.to_json() for field in self.fields],
            }
        )

    @classmethod
    def from_json(cls, json_data: dict) -> ColumnConstraints:
        return cls(
            immutable_required=json_data["immutable_required"],
            partial_updates=(
                json_data["partial_updates"]
                if "partial_updates" in json_data
                else False
            ),
            unique_id_required=json_data["unique_id_required"],
            unique_required=json_data["unique_required"],
            fields=json_data["fields"],
        )


class Column:
    id: uuid.UUID
    table: str
    name: str
    data_type: ResourceID | None
    type: DataType
    is_array: bool
    default_value: str
    search_indexed: bool
    index_type: ColumnIndexType
    constraints: ColumnConstraints

    def __init__(
        self,
        id: uuid.UUID,
        name: str,
        type: str | DataType,
        is_array: bool,
        default_value: str,
        index_type: str | ColumnIndexType,
        search_indexed: bool = False,
        data_type: ResourceID | None = None,
        constraints: ColumnConstraints | None = None,
        table: str = "users",
    ) -> None:
        self.id = id
        self.name = name
        self.data_type = data_type
        self.type = DataType(type)
        self.is_array = is_array
        self.default_value = default_value
        self.search_indexed = search_indexed
        self.index_type = ColumnIndexType(index_type)
        self.table = table
        if constraints is None:
            self.constraints = ColumnConstraints(
                immutable_required=False,
                partial_updates=False,
                unique_id_required=False,
                unique_required=False,
                fields=[],
            )
        else:
            self.constraints = constraints

    def to_json(self) -> str:
        return ucjson.dumps(
            {
                "id": str(self.id),
                "table": self.table,
                "name": self.name,
                "data_type": self.data_type,
                "type": self.type.value,
                "is_array": self.is_array,
                "default_value": self.default_value,
                "search_indexed": self.search_indexed,
                "index_type": self.index_type.value,
                "constraints": self.constraints,
            }
        )

    @classmethod
    def from_json(cls, json_data: dict) -> Column:
        return cls(
            id=uuid.UUID(json_data["id"]),
            table=json_data["table"],
            name=json_data["name"],
            data_type=ResourceID.from_json(json_data["data_type"]),
            type=DataType(json_data["type"]),
            is_array=json_data["is_array"],
            default_value=json_data["default_value"],
            search_indexed=json_data["search_indexed"],
            index_type=ColumnIndexType(json_data["index_type"]),
            constraints=json_data["constraints"],
        )

    def __str__(self) -> str:
        return f"Column {self.table} {self.name} [{self.type}] [{self.data_type}] - {self.id}"

    def __repr__(self) -> str:
        return f"Column(id={self.id}, table={self.table}, name={self.name}, data_type={self.data_type}, type={self.type})"


class Purpose:
    id: uuid.UUID
    name: str
    description: str

    def __init__(self, id: uuid.UUID, name: str, description: str) -> None:
        self.id = id
        self.name = name
        self.description = description

    def to_json(self) -> str:
        return ucjson.dumps(
            {"id": str(self.id), "name": self.name, "description": self.description}
        )

    @classmethod
    def from_json(cls, json_data: dict) -> Purpose:
        return cls(
            id=uuid.UUID(json_data["id"]),
            name=json_data["name"],
            description=json_data["description"],
        )

    def __str__(self) -> str:
        return f"Purpose {self.name} - {self.id}"

    def __repr__(self) -> str:
        return f"Purpose(id={self.id}, name={self.name})"


class ColumnOutputConfig:
    column: ResourceID
    transformer: ResourceID
    token_access_policy: ResourceID | None

    def __init__(
        self,
        column: ResourceID,
        transformer: ResourceID,
        token_access_policy: ResourceID | None = None,
    ) -> None:
        self.column = column
        self.transformer = transformer
        self.token_access_policy = token_access_policy

    def __str__(self) -> str:
        return f"ColumnOutputConfig: {self.column} - {self.transformer} - {self.token_access_policy}"

    def __repr__(self) -> str:
        return f"ColumnOutputConfig(column={self.column!r}, transformer={self.transformer!r}, token_access_policy={self.token_access_policy!r})"

    @classmethod
    def from_json(cls, json_data: dict) -> ColumnOutputConfig:
        return ColumnOutputConfig(
            column=ResourceID.from_json(json_data["column"]),
            transformer=ResourceID.from_json(json_data["transformer"]),
            token_access_policy=ResourceID.from_json(json_data["token_access_policy"]),
        )


class Accessor:
    id: uuid.UUID
    name: str
    description: str
    columns: list[ColumnOutputConfig]
    access_policy: ResourceID
    selector_config: UserSelectorConfig
    purposes: list[ResourceID]
    data_life_cycle_state: DataLifeCycleState
    use_search_index: bool
    version: int

    def __init__(
        self,
        id: uuid.UUID,
        name: str,
        description: str,
        columns: list[ColumnOutputConfig],
        access_policy: ResourceID,
        selector_config: UserSelectorConfig,
        purposes: list[ResourceID],
        data_life_cycle_state: str | DataLifeCycleState = DataLifeCycleState.LIVE,
        use_search_index: bool = False,
        version: int = 0,
    ) -> None:
        self.id = id
        self.name = name
        self.description = description
        self.columns = columns
        self.access_policy = access_policy
        self.selector_config = selector_config
        self.purposes = purposes
        self.data_life_cycle_state = DataLifeCycleState(data_life_cycle_state)
        self.use_search_index = use_search_index
        self.version = version

    def to_json(self) -> str:
        return ucjson.dumps(
            {
                "id": str(self.id),
                "name": self.name,
                "description": self.description,
                "version": self.version,
                "columns": self.columns,
                "access_policy": self.access_policy,
                "selector_config": self.selector_config.to_json(),
                "purposes": self.purposes,
                "data_life_cycle_state": self.data_life_cycle_state.value,
                "use_search_index": self.use_search_index,
            }
        )

    @classmethod
    def from_json(cls, json_data: dict) -> Accessor:
        return cls(
            id=uuid.UUID(json_data["id"]),
            name=json_data["name"],
            description=json_data["description"],
            columns=json_data["columns"],
            access_policy=ResourceID.from_json(json_data["access_policy"]),
            selector_config=UserSelectorConfig.from_json(json_data["selector_config"]),
            purposes=json_data["purposes"],
            data_life_cycle_state=DataLifeCycleState(
                json_data["data_life_cycle_state"]
            ),
            use_search_index=json_data["use_search_index"],
            version=json_data["version"],
        )

    def __str__(self) -> str:
        return f"Accessor - {self.name} - {self.id}"

    def __repr__(self) -> str:
        return f"Accessor(id={self.id}, name={self.name})"


class ColumnInputConfig:
    column: ResourceID
    normalizer: ResourceID

    def __init__(self, column: ResourceID, normalizer: ResourceID) -> None:
        self.column = column
        self.normalizer = normalizer

    def __str__(self) -> str:
        return f"ColumnInputConfig: {self.column} - {self.normalizer}"

    def __repr__(self) -> str:
        return (
            f"ColumnInputConfig(column={self.column!r}, normalizer={self.normalizer!r})"
        )

    @classmethod
    def from_json(cls, json_data: dict) -> ColumnInputConfig:
        return cls(
            column=ResourceID.from_json(json_data["column"]),
            normalizer=ResourceID.from_json(json_data["normalizer"]),
        )


class Mutator:
    id: uuid.UUID
    name: str
    description: str
    columns: list[ColumnInputConfig]
    access_policy: ResourceID
    selector_config: UserSelectorConfig
    version: int

    def __init__(
        self,
        id: uuid.UUID,
        name: str,
        description: str,
        columns: list[ColumnInputConfig],
        access_policy: ResourceID,
        selector_config: UserSelectorConfig,
        version: int = 0,
    ) -> None:
        self.id = id
        self.name = name
        self.description = description
        self.columns = columns
        self.access_policy = access_policy
        self.selector_config = selector_config
        self.version = version

    def to_json(self) -> str:
        return ucjson.dumps(
            {
                "id": str(self.id),
                "name": self.name,
                "description": self.description,
                "version": self.version,
                "columns": self.columns,
                "access_policy": str(self.access_policy),
                "selector_config": self.selector_config.to_json(),
            }
        )

    @classmethod
    def from_json(cls, json_data: dict) -> Mutator:
        return cls(
            id=uuid.UUID(json_data["id"]),
            name=json_data["name"],
            description=json_data["description"],
            columns=[ColumnInputConfig.from_json(j) for j in json_data["columns"]],
            access_policy=ResourceID.from_json(json_data["access_policy"]),
            selector_config=UserSelectorConfig.from_json(json_data["selector_config"]),
            version=json_data["version"],
        )

    def __str__(self) -> str:
        return f"Mutator {self.name} - {self.id}"

    def __repr__(self) -> str:
        return f"Mutator(id={self.id}, name={self.name})"


class AccessPolicyTemplate:
    id: uuid.UUID
    name: str
    description: str
    function: str
    version: int

    def __init__(
        self,
        id: uuid.UUID = uuid.UUID(int=0),
        name: str = "",
        description: str = "",
        function: str = "",
        version: int = 0,
    ) -> None:
        self.id = id
        self.name = name
        self.description = description
        self.function = function
        self.version = version

    def __str__(self) -> str:
        return f"AccessPolicyTemplate {self.name} - {self.id}"

    def __repr__(self) -> str:
        return f"AccessPolicyTemplate(id={self.id}, name={self.name})"

    def to_json(self) -> str:
        return ucjson.dumps(
            {
                "id": str(self.id),
                "name": self.name,
                "description": self.description,
                "function": self.function,
                "version": self.version,
            }
        )

    @classmethod
    def from_json(cls, json_data: dict) -> AccessPolicyTemplate:
        return cls(
            id=uuid.UUID(json_data["id"]),
            name=json_data["name"],
            description=json_data["description"],
            function=json_data["function"],
            version=json_data["version"],
        )


class AccessPolicyComponent:
    def __init__(
        self, policy: str = "", template: str = "", template_parameters: str = ""
    ) -> None:
        if policy != "":
            setattr(self, "policy", policy)
        if template != "":
            setattr(self, "template", template)
            setattr(self, "template_parameters", template_parameters)

    def __str__(self) -> str:
        return f"AccessPolicyComponent: policy:{self.policy} template:{self.template} [{self.template_parameters}]"

    def __repr__(self) -> str:
        if hasattr(self, "policy") and hasattr(self, "template"):
            return f"AccessPolicyComponent(policy={self.policy}, template={self.template}, template_parameters={self.template_parameters})"
        if hasattr(self, "template"):
            return f"AccessPolicyComponent(template={self.template})"
        if hasattr(self, "policy"):
            return f"AccessPolicyComponent(policy={self.policy})"
        return "AccessPolicyComponent()"

    def to_json(self) -> str:
        obj = {}
        if self.policy:
            obj["policy"] = self.policy.to_json()
        if self.template:
            obj["template"] = self.template.to_json()
            obj["template_parameters"] = self.template_parameters
        return ucjson.dumps(obj)

    @classmethod
    def from_json(cls, json_data: dict) -> AccessPolicyComponent:
        return cls(
            policy=json_data["policy"] if "policy" in json_data else "",
            template=json_data["template"] if "template" in json_data else "",
            template_parameters=(
                json_data["template_parameters"]
                if "template_parameters" in json_data
                else ""
            ),
        )


class AccessPolicy:
    id: uuid.UUID
    name: str
    description: str
    policy_type: PolicyType
    version: int
    components: list[AccessPolicyComponent]

    def __init__(
        self,
        id: uuid.UUID = uuid.UUID(int=0),
        name: str = "",
        description: str = "",
        policy_type: str | PolicyType = PolicyType.COMPOSITE_AND,
        version: int = 0,
        components: list[AccessPolicyComponent] = [],
    ) -> None:
        self.id = id
        self.name = name
        self.description = description
        self.policy_type = PolicyType(policy_type)
        self.version = version
        self.components = components

    def __str__(self) -> str:
        return f"AccessPolicy {self.name}[{self.policy_type}] - {self.id}"

    def __repr__(self) -> str:
        return f"AccessPolicy(id={self.id}, name={self.name}, policy_type={self.policy_type}, version={self.version}, components={self.components})"

    def to_json(self) -> str:
        return ucjson.dumps(
            {
                "id": str(self.id),
                "name": self.name,
                "description": self.description,
                "policy_type": self.policy_type.value,
                "version": self.version,
                "components": [c.to_json() for c in self.components],
            }
        )

    @classmethod
    def from_json(cls, json_data: dict) -> AccessPolicy:
        return cls(
            id=uuid.UUID(json_data["id"]),
            name=json_data["name"],
            description=json_data["description"],
            policy_type=PolicyType(json_data["policy_type"]),
            version=json_data["version"],
            components=[
                AccessPolicyComponent.from_json(apc) for apc in json_data["components"]
            ],
        )


class Transformer:
    id: uuid.UUID
    name: str
    input_data_type: ResourceID | None
    input_type: DataType
    input_type_constraints: ColumnConstraints
    output_data_type: ResourceID | None
    output_type: DataType
    output_type_constraints: ColumnConstraints
    reuse_existing_token: bool
    transform_type: TransformType
    function: str
    parameters: str
    version: int

    def __init__(
        self,
        id: uuid.UUID = uuid.UUID(int=0),
        name: str = "",
        input_data_type: ResourceID | None = None,
        input_type: str | DataType = DataType.STRING,
        output_data_type: ResourceID | None = None,
        output_type: str | DataType = DataType.STRING,
        reuse_existing_token: bool = False,
        transform_type: str | TransformType = TransformType.PASSTHROUGH,
        function: str = "",
        parameters: str = "",
        input_type_constraints: ColumnConstraints | None = None,
        output_type_constraints: ColumnConstraints | None = None,
        version: int = 0,
    ) -> None:
        self.id = id
        self.name = name
        self.input_data_type = input_data_type
        self.input_type = DataType(input_type)
        self.output_data_type = output_data_type
        self.output_type = DataType(output_type)
        self.reuse_existing_token = reuse_existing_token
        self.transform_type = TransformType(transform_type)
        self.function = function
        self.parameters = parameters
        if input_type_constraints is None:
            self.input_type_constraints = ColumnConstraints(
                immutable_required=False,
                partial_updates=False,
                unique_id_required=False,
                unique_required=False,
                fields=[],
            )
        else:
            self.input_type_constraints = input_type_constraints
        if output_type_constraints is None:
            self.output_type_constraints = ColumnConstraints(
                immutable_required=False,
                partial_updates=False,
                unique_id_required=False,
                unique_required=False,
                fields=[],
            )
        else:
            self.output_type_constraints = output_type_constraints
        self.version = version

    def __repr__(self) -> str:
        return f"Transformer(id={self.id}, name={self.name}, input_data_type={self.input_data_type}, input_type={self.input_type}, input_type_constraints={self.input_type_constraints}, output_data_type={self.output_data_type}, output_type={self.output_type}, output_type_constraints={self.output_type_constraints}, reuse_existing_token={self.reuse_existing_token}, transform_type={self.transform_type}, function={self.function}, parameters={self.parameters}, version={self.version})"

    def __str__(self) -> str:
        return f"Transformer {self.name} - {self.id}"

    def to_json(self) -> str:
        return ucjson.dumps(
            {
                "id": str(self.id),
                "name": self.name,
                "input_data_type": self.input_data_type,
                "input_type": self.input_type.value,
                "input_type_constraints": self.input_type_constraints,
                "output_data_type": self.output_data_type,
                "output_type": self.output_type.value,
                "output_type_constraints": self.output_type_constraints,
                "reuse_existing_token": self.reuse_existing_token,
                "transform_type": self.transform_type.value,
                "function": self.function,
                "parameters": self.parameters,
                "version": self.version,
            },
        )

    @classmethod
    def from_json(cls, json_data: dict) -> Transformer:
        return cls(
            id=uuid.UUID(json_data["id"]),
            name=json_data["name"],
            input_data_type=ResourceID.from_json(json_data["input_data_type"]),
            input_type=DataType(json_data["input_type"]),
            input_type_constraints=(
                json_data["input_type_constraints"]
                if "input_type_constraints" in json_data
                else None
            ),
            output_data_type=ResourceID.from_json(json_data["output_data_type"]),
            output_type=DataType(json_data["output_type"]),
            output_type_constraints=(
                json_data["output_type_constraints"]
                if "output_type_constraints" in json_data
                else None
            ),
            reuse_existing_token=json_data["reuse_existing_token"],
            transform_type=TransformType(json_data["transform_type"]),
            function=json_data["function"],
            parameters=json_data["parameters"],
            version=json_data["version"],
        )


class RetentionDuration:
    unit: str
    duration: int

    def __init__(self, unit: str, duration: int):
        self.unit = unit
        self.duration = duration

    def __repr__(self) -> str:
        return f"RetentionDuration(unit={self.unit}, duration={self.duration})"

    def __str__(self) -> str:
        return f"RetentionDuration(unit: '{self.unit}', duration: '{self.duration}')"

    def to_json(self) -> str:
        return ucjson.dumps(
            {
                "unit": self.unit,
                "duration": self.duration,
            }
        )

    @classmethod
    def from_json(cls, data: dict) -> RetentionDuration:
        return cls(
            unit=data["unit"],
            duration=data["duration"],
        )


class ColumnRetentionDuration:
    duration_type: DataLifeCycleState
    duration: RetentionDuration
    id: uuid.UUID
    column_id: uuid.UUID
    purpose_id: uuid.UUID
    use_default: bool
    default_duration: RetentionDuration | None
    purpose_name: str
    version: int

    def __init__(
        self,
        duration_type: str | DataLifeCycleState,
        duration: RetentionDuration,
        id: uuid.UUID = uuid.UUID(int=0),
        column_id: uuid.UUID = uuid.UUID(int=0),
        purpose_id: uuid.UUID = uuid.UUID(int=0),
        use_default: bool = False,
        default_duration: RetentionDuration = None,
        purpose_name: str = None,
        version: int = 0,
    ):
        self.duration_type = DataLifeCycleState(duration_type)
        self.id = id
        self.column_id = column_id
        self.purpose_id = purpose_id
        self.duration = duration
        self.use_default = use_default
        self.default_duration = default_duration
        self.purpose_name = purpose_name
        self.version = version

    def __repr__(self):
        return f"ColumnRetentionDuration(duration_type: '{self.duration_type}', duration: '{self.duration}', id: '{self.id}', column_id: '{self.column_id}', purpose_id: '{self.purpose_id}', use_default: '{self.use_default}', default_duration: '{self.default_duration}', purpose_name: '{self.purpose_name}', version: '{self.version}')"

    def to_json(self):
        default_duration = (
            None if self.default_duration is None else self.default_duration.to_json()
        )
        return ucjson.dumps(
            {
                "duration_type": self.duration_type.value,
                "id": str(self.id),
                "column_id": str(self.column_id),
                "purpose_id": str(self.purpose_id),
                "duration": self.duration.to_json(),
                "use_default": self.use_default,
                "default_duration": default_duration,
                "purpose_name": self.purpose_name,
                "version": self.version,
            }
        )

    @classmethod
    def from_json(cls, data):
        return cls(
            DataLifeCycleState(data["duration_type"]),
            RetentionDuration.from_json(data["duration"]),
            uuid.UUID(data["id"]),
            uuid.UUID(data["column_id"]),
            uuid.UUID(data["purpose_id"]),
            data["use_default"],
            RetentionDuration.from_json(data["default_duration"]),
            data["purpose_name"],
            data["version"],
        )


class UpdateColumnRetentionDurationRequest:
    retention_duration: ColumnRetentionDuration

    def __init__(
        self,
        retention_duration: ColumnRetentionDuration,
    ):
        self.retention_duration = retention_duration

    def __repr__(self):
        return f"UpdateColumnRetentionDurationRequest(retention_duration: '{self.retention_duration}')"

    def to_json(self):
        return ucjson.dumps(
            {
                "retention_duration": self.retention_duration,
            }
        )

    @classmethod
    def from_json(cls, data):
        return cls(ColumnRetentionDuration.from_json(data["retention_duration"]))


class UpdateColumnRetentionDurationsRequest:
    retention_durations: list[ColumnRetentionDuration]

    def __init__(
        self,
        retention_durations: list[ColumnRetentionDuration],
    ):
        self.retention_durations = retention_durations

    def __repr__(self):
        return f"UpdateColumnRetentionDurationsRequest(retention_durations: '{self.retention_durations}')"

    def to_json(self):
        return ucjson.dumps(
            {
                "retention_durations": self.retention_durations,
            }
        )

    @classmethod
    def from_json(cls, data):
        return cls(
            [
                ColumnRetentionDuration.from_json(rd)
                for rd in data["retention_durations"]
            ]
        )


class ColumnRetentionDurationResponse:
    max_duration: RetentionDuration
    retention_duration: ColumnRetentionDuration

    def __init__(
        self,
        max_duration: RetentionDuration,
        retention_duration: ColumnRetentionDuration,
    ):
        self.max_duration = max_duration
        self.retention_duration = retention_duration

    def __repr__(self):
        return f"ColumnRetentionDurationResponse(max_duration: '{self.max_duration}', retention_duration: '{self.retention_duration}')"

    @classmethod
    def from_json(cls, data):
        return cls(
            RetentionDuration.from_json(data["max_duration"]),
            ColumnRetentionDuration.from_json(data["retention_duration"]),
        )


class ColumnRetentionDurationsResponse:
    max_duration: RetentionDuration
    retention_durations: list[ColumnRetentionDuration]

    def __init__(
        self,
        max_duration: RetentionDuration,
        retention_durations: list[ColumnRetentionDuration],
    ):
        self.max_duration = max_duration
        self.retention_durations = retention_durations

    def __str__(self):
        return f"ColumnRetentionDurationsResponse(max_duration: '{self.max_duration}', retention_durations: '{self.retention_durations}')"

    def __repr__(self):
        return f"ColumnRetentionDurationsResponse(max_duration={self.max_duration!r}, retention_durations={self.retention_durations!r})"

    @classmethod
    def from_json(cls, data):
        return cls(
            RetentionDuration.from_json(data["max_duration"]),
            [
                ColumnRetentionDuration.from_json(rd)
                for rd in data["retention_durations"]
            ],
        )


class InspectTokenResponse:
    id: uuid.UUID
    token: str

    created: datetime.datetime
    updated: datetime.datetime

    transformer: Transformer
    access_policy: AccessPolicy

    def __init__(
        self,
        id: uuid.UUID,
        token: str,
        created: datetime.datetime,
        updated: datetime.datetime,
        transformer: Transformer,
        access_policy: AccessPolicy,
    ):
        self.id = id
        self.token = token

        self.created = created
        self.updated = updated

        self.transformer = transformer
        self.access_policy = access_policy

    def to_json(self) -> str:
        return ucjson.dumps(
            {
                "id": str(self.id),
                "token": self.token,
                "created": self.created,
                "updated": self.updated,
                "transformer": self.transformer.__dict__,
                "access_policy": self.access_policy.__dict__,
            },
            ensure_ascii=False,
        )

    @classmethod
    def from_json(cls, json_data: dict) -> InspectTokenResponse:
        return cls(
            id=uuid.UUID(json_data["id"]),
            token=json_data["token"],
            created=iso8601.parse_date(json_data["created"]),
            updated=iso8601.parse_date(json_data["updated"]),
            transformer=Transformer.from_json(json_data["transformer"]),
            access_policy=AccessPolicy.from_json(json_data["access_policy"]),
        )


class APIErrorResponse:
    error: str
    id: uuid.UUID
    secondary_id: uuid.UUID
    identical: bool

    def __init__(
        self, error: str, id: uuid.UUID, secondary_id: uuid.UUID, identical: bool
    ):
        self.error = error
        self.id = id
        self.secondary_id = secondary_id
        self.identical = identical

    def to_json(self) -> str:
        return ucjson.dumps(
            {
                "error": self.error,
                "id": self.id,
                "secondary_id": self.secondary_id,
                "identical": self.identical,
            }
        )

    @classmethod
    def from_json(cls, json_data: dict) -> APIErrorResponse:
        return cls(
            error=json_data["error"],
            id=uuid.UUID(json_data["id"]),
            secondary_id=(
                uuid.UUID(json_data["secondary_id"])
                if "secondary_id" in json_data
                else uuid.UUID(int=0)
            ),
            identical=json_data["identical"],
        )


class ColumnConsentedPurposes:
    column: ResourceID
    consented_purposes: list[ResourceID]

    def __init__(
        self, *, column: ResourceID, consented_purposes: list[ResourceID]
    ) -> None:
        self.column = column
        self.consented_purposes = consented_purposes

    @classmethod
    def from_json(cls, data):
        cps = data["consented_purposes"]
        return cls(
            column=ResourceID.from_json(data["column"]),
            consented_purposes=[ResourceID.from_json(cp) for cp in cps],
        )

    def __repr__(self) -> str:
        return f"ColumnConsentedPurposes(column={self.column!r}, consented_purposes={self.consented_purposes!r})"

    def __str__(self) -> str:
        return f"Consented Purposes for {self.column}: {self.consented_purposes})"


@dataclass
class Address:
    """Address dataclass for usercloudssdk - can be used with columns of address type when creating or modifying users"""

    country: str | None = None
    name: str | None = None
    organization: str | None = None
    street_address_line_1: str | None = None
    street_address_line_2: str | None = None
    dependent_locality: str | None = None
    locality: str | None = None
    administrative_area: str | None = None
    post_code: str | None = None
    sorting_code: str | None = None

    @classmethod
    def from_json(cls, json_data: dict) -> Address:
        return cls(
            country=json_data["country"],
            name=json_data["name"],
            organization=json_data["organization"],
            street_address_line_1=json_data["street_address_line_1"],
            street_address_line_2=json_data["street_address_line_2"],
            dependent_locality=json_data["dependent_locality"],
            locality=json_data["locality"],
            administrative_area=json_data["administrative_area"],
            post_code=json_data["post_code"],
            sorting_code=json_data["sorting_code"],
        )

    def to_dict(self) -> dict:
        return {k: v for k, v in asdict(self).items() if v}


@dataclass
class Object:
    id: uuid.UUID
    type_id: uuid.UUID
    alias: str | None = None
    created: datetime.datetime | None = None
    updated: datetime.datetime | None = None
    deleted: datetime.datetime | None = None
    organization_id: uuid.UUID | None = None

    @classmethod
    def from_json(cls, json_data: dict) -> Object:
        return cls(
            id=uuid.UUID(json_data["id"]),
            type_id=uuid.UUID(json_data["type_id"]),
            alias=json_data.get("alias"),
            created=iso8601.parse_date(json_data["created"]),
            updated=iso8601.parse_date(json_data["updated"]),
            deleted=iso8601.parse_date(json_data["deleted"]),
            organization_id=_maybe_get_org_id(json_data),
        )


@dataclass
class Edge:
    id: uuid.UUID
    edge_type_id: uuid.UUID
    source_object_id: uuid.UUID
    target_object_id: uuid.UUID
    created: datetime.datetime | None = None
    updated: datetime.datetime | None = None
    deleted: datetime.datetime | None = None

    @classmethod
    def from_json(cls, json_data: dict) -> Edge:
        return cls(
            id=uuid.UUID(json_data["id"]),
            edge_type_id=uuid.UUID(json_data["edge_type_id"]),
            source_object_id=uuid.UUID(json_data["source_object_id"]),
            target_object_id=uuid.UUID(json_data["target_object_id"]),
            created=iso8601.parse_date(json_data["created"]),
            updated=iso8601.parse_date(json_data["updated"]),
            deleted=iso8601.parse_date(json_data["deleted"]),
        )


@dataclass
class ObjectType:
    id: uuid.UUID
    type_name: str
    created: datetime.datetime | None = None
    updated: datetime.datetime | None = None
    deleted: datetime.datetime | None = None
    organization_id: uuid.UUID | None = None

    @classmethod
    def from_json(cls, json_data: dict) -> ObjectType:
        return cls(
            id=uuid.UUID(json_data["id"]),
            type_name=json_data["type_name"],
            created=iso8601.parse_date(json_data["created"]),
            updated=iso8601.parse_date(json_data["updated"]),
            deleted=iso8601.parse_date(json_data["deleted"]),
            organization_id=_maybe_get_org_id(json_data),
        )


@dataclass
class Attribute:
    name: str
    direct: bool
    inherit: bool
    propagate: bool

    @classmethod
    def from_json(cls, json_data: dict) -> Attribute:
        return cls(
            name=json_data["name"],
            direct=json_data["direct"],
            inherit=json_data["inherit"],
            propagate=json_data["propagate"],
        )


@dataclass
class EdgeType:
    id: uuid.UUID
    type_name: str
    source_object_type_id: uuid.UUID
    target_object_type_id: uuid.UUID
    attributes: list[Attribute]
    created: datetime.datetime | None = None
    updated: datetime.datetime | None = None
    deleted: datetime.datetime | None = None
    organization_id: uuid.UUID | None = None

    @classmethod
    def from_json(cls, json_data: dict) -> EdgeType:
        return cls(
            id=uuid.UUID(json_data["id"]),
            type_name=json_data["type_name"],
            source_object_type_id=uuid.UUID(json_data["source_object_type_id"]),
            target_object_type_id=uuid.UUID(json_data["target_object_type_id"]),
            attributes=[
                Attribute.from_json(attr) for attr in json_data["attributes"] or []
            ],
            created=iso8601.parse_date(json_data["created"]),
            updated=iso8601.parse_date(json_data["updated"]),
            deleted=iso8601.parse_date(json_data["deleted"]),
            organization_id=_maybe_get_org_id(json_data),
        )


@dataclass
class Organization:
    id: uuid.UUID
    name: str
    region: str
    created: datetime.datetime | None = None
    updated: datetime.datetime | None = None
    deleted: datetime.datetime | None = None

    @classmethod
    def from_json(cls, json_data: dict) -> Organization:
        return cls(
            id=uuid.UUID(json_data["id"]),
            name=json_data["name"],
            region=json_data["region"],
            created=iso8601.parse_date(json_data["created"]),
            updated=iso8601.parse_date(json_data["updated"]),
            deleted=iso8601.parse_date(json_data["deleted"]),
        )


def _maybe_get_org_id(json_data: dict) -> uuid.UUID | None:
    return _uuid_or_none(json_data, "organization_id")


def _uuid_or_none(json_data: dict, field: str) -> uuid.UUID | None:
    return uuid.UUID(json_data[field]) if field in json_data else None
