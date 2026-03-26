package utils

// ToPtr 将值转换为指针
func ToPtr[T any](v T) *T {
	return &v
}
