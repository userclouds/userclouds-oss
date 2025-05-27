package messageelements

// ElementType is the type of configurable element
// in a user-facing message. These elements can be customized
// by app, and we provide defaults by MessageType
type ElementType string

// ElementsByElementType is a map from element type to element
type ElementsByElementType map[ElementType]string

// ElementGetter is a function type that returns the appropriate
// message element for a passed in ElementType. ElementGetters are
// used by email.SendWithHTMLTemplate and other other callers
type ElementGetter func(ElementType) string

// MessageType is the type of user-facing message being sent,
// used for looking up the appropriate message elements for
// an ElementType and app
type MessageType string

// ElementsByElementTypeByMessageType is a map from message type to element type to element
type ElementsByElementTypeByMessageType map[MessageType]ElementsByElementType
