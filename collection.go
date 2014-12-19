package cmagic

import (
	"encoding/json"
	"errors"
	g "github.com/hailocab/cmagic/generate"
	r "github.com/hailocab/cmagic/reflect"
	"reflect"
)

type table struct {
	nameSpace      *nameSpace
	TableInfo *TableInfo
}

// Contains mostly analyzed information about the entity
type tableInfo struct {
	keyspace, name string
	entity         interface{}
	keys 		   Keys
	fieldNames     map[string]struct{} // This is here only to check containment
	fields         []string
	fieldValues    []interface{}
}

func newTableInfo(keyspace, name, keys Keys, entity interface{}) *tableInfo {
	cinf := &tableInfo{
		keyspace:   keyspace,
		name:       name,
		entity:     entity,
		primaryKey: primaryKey,
	}
	fields, values, ok := r.FieldsAndValues(entity)
	if !ok {
		// panicking here since this is a programmer error
		panic("Supplied entity is not a struct")
	}
	cinf.fieldNames = map[string]struct{}{}
	for _, v := range fields {
		if v == cinf.primaryKey {
			continue
		}
		cinf.fieldNames[v] = struct{}{}
	}
	cinf.fields = fields
	cinf.fieldValues = values
	return cinf
}

func (c Table) zero() interface{} {
	return reflect.New(reflect.TypeOf(c.TableInfo.entity)).Interface()
}

// Since we cant have Map -> [(k, v)] we settle for Map -> ([k], [v])
// #tupleLessLifeSucks
func keyValues(m map[string]interface{}) ([]string, []interface{}) {
	keys := []string{}
	values := []interface{}{}
	for k, v := range m {
		keys = append(keys, k)
		values = append(values, v)
	}
	return keys, values
}

func toMap(i interface{}) (map[string]interface{}, bool) {
	switch v := i.(type) {
	case M:
		return map[string]interface{}(v), true
	case map[string]interface{}:
		return v, true
	}
	return r.StructToMap(i)
}

// Will return 'entity' struct what was supplied when initializing the Table
func (c table) Read(id string) (interface{}, error) {
	stmt := g.ReadById(c.nameSpace.name, c.TableInfo.primaryKey)
	m := map[string]interface{}{}
	sess := c.nameSpace.session
	sess.Query(stmt, id).Iter().MapScan(m)
	bytes, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	ret := c.zero()
	err = json.Unmarshal(bytes, ret)
	return ret, err
}

func (c table) Create(i interface{}) error {
	m, ok := toMap(i)
	if !ok {
		return errors.New("Can't create: value not understood")
	}
	fields, values := keyValues(m)
	stmt := g.Insert(c.TableInfo.name, fields)
	sess := c.nameSpace.session
	return sess.Query(stmt, values...).Exec()
}

// Use structs for the time being, no maps please.
func (c table) Update(i interface{}) error {
	m, ok := toMap(i)
	if !ok {
		return errors.New("Update: value not understood")
	}
	id, ok := m[c.TableInfo.primaryKey]
	if !ok {
		return errors.New("Update: primary key not found")
	}
	fields, values := keyValues(m)
	for k, v := range m {
		if k == c.TableInfo.primaryKey {
			continue
		}
		fields = append(fields, k)
		values = append(values, v)
	}
	stmt := g.UpdateById(c.nameSpace.name, c.TableInfo.primaryKey, fields)
	sess := c.nameSpace.session
	return sess.Query(stmt, append(values, id)...).Exec()
}

func (c table) ReadOpt(id string, opt *RowOptions) (interface{}, error) {
	return nil, errors.New("ReadOpt not implemented yet")
}

func (c table) Delete(id string) error {
	return c.nameSpace.session.Query(g.DeleteById(c.nameSpace.name, c.TableInfo.primaryKey), id).Exec()
}
