package gorm_generics

type GormModel[E any] interface {
	ToEntity() E
	FromEntity(entity E) interface{}
}
