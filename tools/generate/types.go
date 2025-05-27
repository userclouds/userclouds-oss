package generate

import "go/types"

// HasMethod returns true if the given type has a method with the given name
func HasMethod(typ types.Type, methodName string) bool {
	for _, t := range []types.Type{typ, types.NewPointer(typ)} {
		ms := types.NewMethodSet(t)
		for i := range ms.Len() {
			if ms.At(i).Obj().Name() == methodName {
				return true
			}
		}
	}
	return false
}
