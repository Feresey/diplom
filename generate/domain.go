package generate

import (
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type Domain interface {
	Next() bool
	Value() string
	Reset()
}

type IntDomain struct {
	counter int
	value   int
	top     int
}

func (id *IntDomain) Next() bool {
	id.counter++
	if id.counter%2 == 0 {
		id.value = id.counter / 2
	} else {
		id.value = -1 * (id.counter / 2)
	}
	return id.value <= id.top
}

func (id *IntDomain) Value() string { return strconv.Itoa(id.value) }

func (id *IntDomain) Reset() {
	id.counter = 0
	id.value = 0
}

func (id *IntDomain) Init(top int) {
	id.Reset()
	id.top = top
}

type FloatDomain struct {
	index int
	value float64
	step  float64
	top   float64
}

func (fd *FloatDomain) Next() bool {
	fd.index++
	if fd.index == 0 {
		// return zero
		return true
	}

	if fd.index%2 == 0 {
		fd.value = -fd.value
	} else {
		fd.value = math.Abs(fd.value) + fd.step
	}

	return fd.value <= fd.top
}

func (fd *FloatDomain) Value() string { return strconv.FormatFloat(fd.value, 'f', -1, 64) }

func (fd *FloatDomain) Reset() {
	fd.index = -1
	fd.value = 0
}

func (fd *FloatDomain) Init(step, top float64) {
	fd.Reset()
	fd.step = step
	fd.top = top
}

type TimeDomain struct {
	index int
	now   time.Time
	value time.Time
	top   int
}

func (td *TimeDomain) Next() bool {
	td.index++
	if td.index == 0 {
		// send now
		return true
	}
	if td.index%2 == 0 {
		td.value = td.value.AddDate(0, 0, -td.index).Add(-time.Second * time.Duration(td.index))
	} else {
		td.value = td.value.AddDate(0, 0, td.index).Add(time.Second * time.Duration(td.index))
	}

	return td.index < td.top
}

func (td *TimeDomain) Value() string { return td.value.Format(time.RFC3339) }
func (td *TimeDomain) Reset()        { td.index = -1 }

func (td *TimeDomain) Init(now time.Time, top int) {
	td.Reset()
	td.top = top
	td.value = now
	td.now = now
}

type UUIDDomain struct{}

func (ud *UUIDDomain) Next() bool    { return true }
func (ud *UUIDDomain) Value() string { return uuid.NewString() }
func (ud *UUIDDomain) Reset()        {}

type EnumDomain struct {
	values []string
	index  int
}

func (ed *EnumDomain) Next() bool {
	ed.index++
	return ed.index < len(ed.values)
}

func (ed *EnumDomain) Value() string { return ed.values[ed.index] }
func (ed *EnumDomain) Reset()        { ed.index = -1 }

func BoolDomain() *EnumDomain {
	return &EnumDomain{
		values: []string{"True", "False"},
		index:  -1,
	}
}
