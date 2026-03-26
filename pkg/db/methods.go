package db

import (
	"errors"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func (p *Postgre) CreateClient(connstrin string) (err error) {
	p.GormDriver, err = gorm.Open(postgres.Open(connstrin), &gorm.Config{})
	if err != nil {
		return
	}
	return
}
func (p *Postgre) Insert(args any) (err error) {
	tx := p.GormDriver.Create(args)
	if tx.Error != nil {
		return
	}
	return
}

func (p *Postgre) ReadOne(args interface{}, conds ...interface{}) (err error) {
	tx := p.GormDriver.First(&args, conds...)
	if tx.Error != nil {
		return
	}
	return
}
func (p *Postgre) UpdateOne(model any, updatevalue any, query string, args ...any) (ack int64, err error) {
	tx := p.GormDriver.Model(&model).Where(query, args...).Updates(updatevalue)
	if tx.Error != nil {
		return
	}
	return tx.RowsAffected, tx.Error
}
func (p *Postgre) DeleteOne(model any, query string, args ...any) (ack int64, err error) {
	tx := p.GormDriver.Where(query, args...).Delete(model)
	return tx.RowsAffected, tx.Error
}
func (p *Postgre) AutoMigrate(model any) (err error) {
	err = p.GormDriver.AutoMigrate(model)
	if err != nil {
		return
	}
	return
}
func (p *Postgre) CheckIfExist(model any, query string, args ...any) (flag bool, err error) {
	count := int64(0)
	err = p.GormDriver.Model(&model).Where(query, args...).Count(&count).Error
	if err != nil {
		err = errors.New("failed to query database")
		return
	}
	return count > 0, nil
}
func (p *Postgre) ReadMany(limit int, offset int, data any, cond ...any) (err error) {
	tx := p.GormDriver.Limit(limit).Offset(offset).Find(&data, cond...)
	if tx.Error != nil {
		return
	}
	return
}
