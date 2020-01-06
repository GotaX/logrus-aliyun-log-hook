package validator

import (
	"fmt"
	"strings"
	"time"
)

func IllegalArgument(field, desc string) error {
	return fmt.Errorf("invalid config %q %v", field, desc)
}

func Required(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return IllegalArgument(field, "is required")
	}
	return nil
}

func All(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func CoalesceStr(strs ...string) (str string) {
	for _, str = range strs {
		if strings.TrimSpace(str) != "" {
			break
		}
	}
	return
}

func CoalesceInt(nums ...int) (num int) {
	for _, num = range nums {
		if num > 0 {
			break
		}
	}
	return
}

func CoalesceDur(durs ...time.Duration) (dur time.Duration) {
	for _, dur = range durs {
		if dur > 0 {
			break
		}
	}
	return
}
