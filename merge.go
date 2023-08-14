package merge

import (
	"fmt"
	"reflect"
)

type Merger[T any] interface {
	Merge(objs ...T) (dst T, err error)
}

type merge[T any] struct {
	pointerMerge func(m *merge[T], dst, src reflect.Value, visited map[uintptr]bool, fieldPath string) (err error)
	structMerge  func(m *merge[T], dst, src reflect.Value, visited map[uintptr]bool, fieldPath string) (err error)
	sliceMerge   func(m *merge[T], dst, src reflect.Value, visited map[uintptr]bool, fieldPath string) (err error)
	arrayMerge   func(m *merge[T], dst, src reflect.Value, visited map[uintptr]bool, fieldPath string) (err error)
	mapMerge     func(m *merge[T], dst, src reflect.Value, visited map[uintptr]bool, fieldPath string) (err error)
	defaultMerge func(m *merge[T], dst, src reflect.Value, visited map[uintptr]bool, fieldPath string) (err error)
}

type Option[T any] func(*merge[T])

func WithPointerFunc[T any](f func(m *merge[T], dst, src reflect.Value, visited map[uintptr]bool, fieldPath string) (err error)) Option[T] {
	return func(m *merge[T]) {
		m.pointerMerge = f
	}
}

func WithStructFunc[T any](f func(m *merge[T], dst, src reflect.Value, visited map[uintptr]bool, fieldPath string) (err error)) Option[T] {
	return func(m *merge[T]) {
		m.structMerge = f
	}
}

func WithSliceFunc[T any](f func(m *merge[T], dst, src reflect.Value, visited map[uintptr]bool, fieldPath string) (err error)) Option[T] {
	return func(m *merge[T]) {
		m.sliceMerge = f
	}
}

func WithArrayFunc[T any](f func(m *merge[T], dst, src reflect.Value, visited map[uintptr]bool, fieldPath string) (err error)) Option[T] {
	return func(m *merge[T]) {
		m.arrayMerge = f
	}
}

func WithMapFunc[T any](f func(m *merge[T], dst, src reflect.Value, visited map[uintptr]bool, fieldPath string) (err error)) Option[T] {
	return func(m *merge[T]) {
		m.mapMerge = f
	}
}

func WithDefaultFunc[T any](f func(m *merge[T], dst, src reflect.Value, visited map[uintptr]bool, fieldPath string) (err error)) Option[T] {
	return func(m *merge[T]) {
		m.defaultMerge = f
	}
}

func (m *merge[T]) deepMerge(dst, src reflect.Value, visited map[uintptr]bool, fieldPath string) (err error) {
	if !src.IsValid() || src.IsZero() {
		return nil
	}
	dType := dst.Type()
	sType := src.Type()
	if dType != sType {
		switch {
		case src.Kind() == reflect.Ptr && dType == src.Elem().Type():
			src = src.Elem()
		case dst.Kind() == reflect.Ptr && dst.Elem().Type() == sType:
			dst = dst.Elem()
		default:
			return fmt.Errorf("types do not match at %s: %v vs %v and %v vs %v", fieldPath, dType, sType, src.Kind(), dst.Kind())
		}
	}
	sKind := src.Kind()
	if sKind == reflect.Ptr {
		src = src.Elem()
	}
	if sKind == reflect.Struct && src.CanAddr() {
		addr := src.Addr().Pointer()
		if visited[addr] {
			return nil
		}
		visited[addr] = true
	}
	switch dst.Kind() {
	case reflect.Ptr:
		return m.pointerMerge(m, dst, src, visited, fieldPath)
	case reflect.Struct:
		return m.structMerge(m, dst, src, visited, fieldPath)
	case reflect.Slice:
		return m.sliceMerge(m, dst, src, visited, fieldPath)
	case reflect.Array:
		return m.arrayMerge(m, dst, src, visited, fieldPath)
	case reflect.Map:
		return m.mapMerge(m, dst, src, visited, fieldPath)
	default:
		return m.defaultMerge(m, dst, src, visited, fieldPath)
	}
}

func New[T any](opts ...Option[T]) Merger[T] {
	m := new(merge[T])
	for _, opt := range append([]Option[T]{
		WithPointerFunc(func(m *merge[T], dst, src reflect.Value, visited map[uintptr]bool, fieldPath string) (err error) {
			return m.deepMerge(dst.Elem(), src, visited, fieldPath)
		}),
		WithStructFunc(func(m *merge[T], dst, src reflect.Value, visited map[uintptr]bool, fieldPath string) (err error) {
			dnum := dst.NumField()
			snum := src.NumField()
			if dnum != snum {
				return fmt.Errorf("number of fields do not match at %s, dst: %d, src: %d", fieldPath, dnum, snum)
			}
			dType := dst.Type()
			for i := 0; i < dnum; i++ {
				dstField := dst.Field(i)
				if dstField.CanSet() {
					nf := fmt.Sprintf("%s.%s(%d)", fieldPath, dType.Field(i).Name, i)
					if err = m.deepMerge(dstField, src.Field(i), visited, nf); err != nil {
						return fmt.Errorf("error in field at %s: %w", nf, err)
					}
				}
			}
			return nil
		}),
		WithSliceFunc(func(m *merge[T], dst, src reflect.Value, visited map[uintptr]bool, fieldPath string) (err error) {
			srcLen := src.Len()
			if srcLen > 0 {
				if dst.IsNil() {
					dst.Set(reflect.MakeSlice(dst.Type(), srcLen, srcLen))
				} else {
					diffLen := srcLen - dst.Len()
					if diffLen > 0 {
						dst.Set(reflect.AppendSlice(dst, reflect.MakeSlice(dst.Type(), diffLen, diffLen)))
					}
				}
				for i := 0; i < srcLen; i++ {
					nf := fmt.Sprintf("%s[%d]", fieldPath, i)
					if err = m.deepMerge(dst.Index(i), src.Index(i), visited, nf); err != nil {
						return fmt.Errorf("error in slice at %s: %w", nf, err)
					}
				}
			}
			return nil
		}),
		WithArrayFunc(func(m *merge[T], dst, src reflect.Value, visited map[uintptr]bool, fieldPath string) (err error) {
			srcLen := src.Len()
			if srcLen != dst.Len() {
				return fmt.Errorf("array lengths do not match at %s, dst: %d, src: %d", fieldPath, dst.Len(), srcLen)
			}
			for i := 0; i < srcLen; i++ {
				nf := fmt.Sprintf("%s[%d]", fieldPath, i)
				if err = m.deepMerge(dst.Index(i), src.Index(i), visited, nf); err != nil {
					return fmt.Errorf("error in array at %s: %w", nf, err)
				}
			}
			return nil
		}),
		WithMapFunc(func(m *merge[T], dst, src reflect.Value, visited map[uintptr]bool, fieldPath string) (err error) {
			dType := dst.Type()
			if dst.IsNil() {
				dst.Set(reflect.MakeMapWithSize(dType, src.Len()))
			}
			dElem := dType.Elem()
			for _, key := range src.MapKeys() {
				vdst := dst.MapIndex(key)
				if !vdst.IsValid() {
					vdst = reflect.New(dElem).Elem()
				}
				nf := fmt.Sprintf("%s[%s]", fieldPath, key)
				if err = m.deepMerge(vdst, src.MapIndex(key), visited, nf); err != nil {
					return fmt.Errorf("error in map at %s: %w", nf, err)
				}
				dst.SetMapIndex(key, vdst)
			}
			return nil
		}),
		WithDefaultFunc(func(m *merge[T], dst, src reflect.Value, visited map[uintptr]bool, fieldPath string) (err error) {
			if dst.CanSet() {
				dst.Set(src)
			}
			return nil
		}),
	}, opts...) {
		opt(m)
	}
	return m
}

func (m *merge[T]) Merge(objs ...T) (dst T, err error) {
	switch len(objs) {
	case 0:
		return dst, nil
	case 1:
		dst = objs[0]
		return dst, nil
	default:
		dst = objs[0]
		visited := make(map[uintptr]bool)
		rdst := reflect.ValueOf(&dst)
		for _, src := range objs[1:] {
			err = m.deepMerge(rdst, reflect.ValueOf(&src), visited, "")
			if err != nil {
				return dst, err
			}
		}
	}
	return dst, err
}

func Merge[T any](dst, src T, opts ...Option[T]) (T, error) {
	return New(opts...).Merge(dst, src)
}
