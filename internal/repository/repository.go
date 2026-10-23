package repository

import "gorm.io/gorm"

type Repository[T any] interface {
	Create(db *gorm.DB, entity *T) error
	Update(db *gorm.DB, entity *T, id any) error
	Delete(db *gorm.DB, id any) error
	GetByID(db *gorm.DB, id any) (*T, error)
}

type repositoryImpl[T any] struct{}

func newRepositoryImpl[T any]() Repository[T] {
	return &repositoryImpl[T]{}
}

func (r *repositoryImpl[T]) Create(db *gorm.DB, entity *T) error {
	return db.Create(entity).Error
}

func (r *repositoryImpl[T]) Update(db *gorm.DB, entity *T, id any) error {
	return db.Model(new(T)).Where("id = ?", id).Updates(entity).Error
}

func (r *repositoryImpl[T]) Delete(db *gorm.DB, id any) error {
	return db.Delete(new(T), id).Error
}

func (r *repositoryImpl[T]) GetByID(db *gorm.DB, id any) (*T, error) {
	var entity T
	return &entity, db.Where("id = ?", id).First(&entity).Error
}