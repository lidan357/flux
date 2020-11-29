package ext

import (
	"github.com/bytepowered/flux"
	"github.com/bytepowered/flux/pkg"
	"sort"
)

type filterWrapper struct {
	filter flux.Filter
	order  int
}

type filterArray []filterWrapper

func (s filterArray) Len() int           { return len(s) }
func (s filterArray) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s filterArray) Less(i, j int) bool { return s[i].order < s[j].order }

var (
	_globalFilter    = make([]filterWrapper, 0, 16)
	_selectiveFilter = make([]filterWrapper, 0, 16)
)

// StoreGlobalFilter 注册全局Filter；
func StoreGlobalFilter(v interface{}) {
	_globalFilter = _checkedAppendFilter(v, _globalFilter)
	sort.Sort(filterArray(_globalFilter))
}

// StoreSelectiveFilter 注册可选Filter；
func StoreSelectiveFilter(v interface{}) {
	_selectiveFilter = _checkedAppendFilter(v, _selectiveFilter)
	sort.Sort(filterArray(_selectiveFilter))
}

func _checkedAppendFilter(v interface{}, in []filterWrapper) (out []filterWrapper) {
	f := pkg.RequireNotNil(v, "Not a valid Filter").(flux.Filter)
	return append(in, filterWrapper{filter: f, order: orderOf(v)})
}

// LoadSelectiveFilters 获取已排序的Filter列表
func LoadSelectiveFilters() []flux.Filter {
	return _getFilters(_selectiveFilter)
}

// LoadGlobalFilters 获取已排序的全局Filter列表
func LoadGlobalFilters() []flux.Filter {
	return _getFilters(_globalFilter)
}

func _getFilters(in []filterWrapper) []flux.Filter {
	out := make([]flux.Filter, len(in))
	for i, v := range in {
		out[i] = v.filter
	}
	return out
}

// LoadSelectiveFilter 获取已排序的可选Filter列表
func LoadSelectiveFilter(filterId string) (flux.Filter, bool) {
	filterId = pkg.RequireNotEmpty(filterId, "filterId is empty")
	for _, f := range _selectiveFilter {
		if filterId == f.filter.TypeId() {
			return f.filter, true
		}
	}
	return nil, false
}

func orderOf(v interface{}) int {
	if v, ok := v.(flux.Orderer); ok {
		return v.Order()
	} else {
		return 0
	}
}
