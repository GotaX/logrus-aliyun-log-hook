package validator

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAll(t *testing.T) {
	err0 := errors.New("e0")
	err1 := errors.New("e1")
	assert.NoError(t, All(nil, nil))
	assert.Error(t, All(err0, nil), err0)
	assert.Error(t, All(nil, err0), err0)
	assert.Error(t, All(err0, err1), err0)
}

func TestRequired(t *testing.T) {
	field, value := "f1", "v1"
	assert.NoError(t, Required(field, value))
	assert.Error(t, Required(field, ""))
}

func TestIllegalArgument(t *testing.T) {
	field, value := "f1", "v1"
	assert.Error(t, IllegalArgument(field, value))
}

func TestCoalesceStr(t *testing.T) {
	s0, s1 := "s0", "s1"
	assert.Equal(t, CoalesceStr(s0, s1), s0)
	assert.Equal(t, CoalesceStr("", s1), s1)
	assert.Equal(t, CoalesceStr("\t", s1), s1)
}

func TestCoalesceInt(t *testing.T) {
	n1, n2 := 1, 2
	assert.Equal(t, CoalesceInt(n1, n2), n1)
	assert.Equal(t, CoalesceInt(0, n2), n2)
	assert.Equal(t, CoalesceInt(-1, n2), n2)
}

func TestCoalesceDur(t *testing.T) {
	t1, t2 := time.Second, 2*time.Second
	assert.Equal(t, CoalesceDur(t1, t2), t1)
	assert.Equal(t, CoalesceDur(0, t2), t2)
	assert.Equal(t, CoalesceDur(-1*time.Second, t2), t2)
}
