package gorm_generics

import (
	"context"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"sync"
)

type GormModel[E any] interface {
	ToEntity() E
	FromEntity(entity E) interface{}
}

func NewRepository[M GormModel[E], E any](db *gorm.DB) *GormRepository[M, E] {
	return &GormRepository[M, E]{
		db: db,
	}
}

type GormRepository[M GormModel[E], E any] struct {
	db                   *gorm.DB
	preloadAssocationsOk bool
	preloadAssocations   bool
	associations         []string
}

func (r *GormRepository[M, E]) EnablePreloadAssociations() *GormRepository[M, E] {
	r.preloadAssocations = true
	return r
}
func (r *GormRepository[M, E]) DisablePreloadAssociations() *GormRepository[M, E] {
	r.preloadAssocations = true
	return r
}

func (r *GormRepository[M, E]) SetPreloadAssociations(association bool) *GormRepository[M, E] {
	if association {
		r.EnablePreloadAssociations()
	} else {
		r.DisablePreloadAssociations()
	}
	return r
}

func (r *GormRepository[M, E]) setAssociations(model *M) *GormRepository[M, E] {
	schema, _ := schema.Parse(model, &sync.Map{}, r.db.NamingStrategy)
	for _, i := range schema.Relationships.Many2Many {
		r.associations = append(r.associations, i.Name)
	}
	return r
}

func (r *GormRepository[M, E]) Insert(ctx context.Context, entity *E) error {
	var start M
	model := start.FromEntity(*entity).(M)

	err := r.db.WithContext(ctx).Create(&model).Error
	if err != nil {
		return err
	}

	*entity = model.ToEntity()
	return nil
}

func (r *GormRepository[M, E]) Delete(ctx context.Context, entity *E) error {
	var start M
	model := start.FromEntity(*entity).(M)
	err := r.db.WithContext(ctx).Delete(model).Error
	if err != nil {
		return err
	}
	return nil
}

func (r *GormRepository[M, E]) DeleteById(ctx context.Context, id any) error {
	var start M
	err := r.db.WithContext(ctx).Delete(&start, &id).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *GormRepository[M, E]) Update(ctx context.Context, entity *E) error {
	var start M
	model := start.FromEntity(*entity).(M)

	err := r.db.WithContext(ctx).Save(&model).Error
	if err != nil {
		return err
	}

	*entity = model.ToEntity()
	return nil
}

func (r *GormRepository[M, E]) FindByID(ctx context.Context, id any) (E, error) {
	var model M
	err := r.db.WithContext(ctx).First(&model, id).Error
	if err != nil {
		return *new(E), err
	}

	return model.ToEntity(), nil
}

func (r *GormRepository[M, E]) Find(ctx context.Context, specifications ...Specification) ([]E, error) {
	return r.FindWithLimit(ctx, -1, -1, specifications...)
}

func (r *GormRepository[M, E]) Count(ctx context.Context, specifications ...Specification) (i int64, err error) {
	model := new(M)
	err = r.getPreWarmDbForSelect(ctx, specifications...).Model(model).Count(&i).Error
	return
}

func (r *GormRepository[M, E]) getPreWarmDbForSelect(ctx context.Context, specification ...Specification) *gorm.DB {
	dbPrewarm := r.db.WithContext(ctx)
	for _, s := range specification {
		dbPrewarm = dbPrewarm.Where(s.GetQuery(), s.GetValues()...)
	}
	return dbPrewarm
}
func (r *GormRepository[M, E]) FindWithLimit(ctx context.Context, limit int, offset int, specifications ...Specification) ([]E, error) {
	var models []M

	dbPrewarm := r.getPreWarmDbForSelect(ctx, specifications...)

	if r.preloadAssocations {
		if !r.preloadAssocationsOk {
			var start M
			r.setAssociations(&start)
		}
		for _, association := range r.associations {
			dbPrewarm = dbPrewarm.Preload(association)
		}
	}

	err := dbPrewarm.Limit(limit).Offset(offset).Find(&models).Error

	if err != nil {
		return nil, err
	}

	result := make([]E, 0, len(models))
	for _, row := range models {
		result = append(result, row.ToEntity())
	}

	return result, nil
}

func (r *GormRepository[M, E]) FindAll(ctx context.Context) ([]E, error) {
	return r.FindWithLimit(ctx, -1, -1)
}
