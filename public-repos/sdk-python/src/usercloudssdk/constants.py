from enum import Enum, unique

MUTATOR_COLUMN_DEFAULT_VALUE = "UCDEF-7f55f479-3822-4976-a8a9-b789d5c6f152"
MUTATOR_COLUMN_CURRENT_VALUE = "UCCUR-7f55f479-3822-4976-a8a9-b789d5c6f152"

_JSON_CONTENT_TYPE = "application/json"

PAGINATION_CURSOR_BEGIN = ""
PAGINATION_CURSOR_END = "end"

PAGINATION_SORT_ASCENDING = "ascending"
PAGINATION_SORT_DESCENDING = "descending"


@unique
class AuthnType(Enum):
    PASSWORD = "password"
    OIDC = "social"


@unique
class ColumnIndexType(Enum):
    NONE = "none"
    INDEXED = "indexed"
    UNIQUE = "unique"


@unique
class DataLifeCycleState(Enum):
    LIVE = "live"
    SOFT_DELETED = "softdeleted"
    POST_DELETE = "postdelete"
    PRE_DELETE = "predelete"


@unique
class DataType(Enum):
    ADDRESS = "address"
    BIRTHDATE = "birthdate"
    BOOLEAN = "boolean"
    COMPOSITE = "composite"
    DATE = "date"
    EMAIL = "email"
    INTEGER = "integer"
    PHONENUMBER = "phonenumber"
    E164_PHONENUMBER = "e164_phonenumber"
    SSN = "ssn"
    STRING = "string"
    TIMESTAMP = "timestamp"
    UUID = "uuid"


@unique
class PolicyType(Enum):
    COMPOSITE_AND = "composite_and"
    COMPOSITE_OR = "composite_or"


@unique
class Region(Enum):
    AWS_US_EAST_1 = "aws-us-east-1"
    AWS_US_WEST_2 = "aws-us-west-2"
    AWS_EU_WEST_1 = "aws-eu-west-1"


@unique
class TransformType(Enum):
    PASSTHROUGH = "passthrough"
    TOKENIZE_BY_REFERENCE = "tokenizebyreference"
    TOKENIZE_BY_VALUE = "tokenizebyvalue"
    TRANSFORM = "transform"
