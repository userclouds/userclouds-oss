package types

// ConfirmOperationFunc can be used to define a user facing prompt for dangerous operations
type ConfirmOperationFunc func(prompt string) bool

// ConfirmOperation by default always returns false but can be used to define a user facing
// prompt for confirming dangerous operations
var ConfirmOperation ConfirmOperationFunc = func(prompt string) bool { return false }

// ConfirmOperationForProd by default always returns false, but can be used to display an
// extra prompt for confirming dangerous operations in production
var ConfirmOperationForProd ConfirmOperationFunc = func(prompt string) bool { return false }
