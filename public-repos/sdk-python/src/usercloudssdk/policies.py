import uuid

from .models import ResourceID

# Note: these need to stay in sync with authz/constants.go
AccessPolicyOpen = ResourceID(id=uuid.UUID("3f380e42-0b21-4570-a312-91e1b80386fa"))
TransformerUUID = ResourceID(id=uuid.UUID("e3743f5b-521e-4305-b232-ee82549e1477"))
TransformerEmail = ResourceID(id=uuid.UUID("0cedf7a4-86ab-450a-9426-478ad0a60faa"))
TransformerFullName = ResourceID(id=uuid.UUID("b9bf352f-b1ee-4fb2-a2eb-d0c346c6404b"))
TransformerSSN = ResourceID(id=uuid.UUID("3f65ee22-2241-4694-bbe3-72cefbe59ff2"))
TransformerCreditCard = ResourceID(id=uuid.UUID("618a4ae7-9979-4ee8-bac5-db87335fe4d9"))
TransformerPassThrough = ResourceID(
    id=uuid.UUID("c0b5b2a1-0b1f-4b9f-8b1a-1b1f4b9f8b1a")
)
NormalizerOpen = ResourceID(id=uuid.UUID("c0b5b2a1-0b1f-4b9f-8b1a-1b1f4b9f8b1a"))
