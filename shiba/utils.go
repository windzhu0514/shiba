package shiba

import (
	"reflect"
	"time"

	"github.com/robfig/cron/v3"
)

// 获取结构体db tag的值列表
func DBFields(rv reflect.Value) []string {
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	rt := rv.Type()

	var fields []string
	if rv.Kind() == reflect.Struct {
		for i := 0; i < rv.NumField(); i++ {
			sf := rv.Field(i)
			if sf.Kind() == reflect.Struct {
				fields = append(fields, DBFields(sf)...)
				continue
			}

			tagName := rt.Field(i).Tag.Get("db")
			if tagName != "" {
				fields = append(fields, tagName)
			}
		}
		return fields
	}

	if rv.Kind() == reflect.Map {
		for _, key := range rv.MapKeys() {
			fields = append(fields, key.String())
		}
		return fields
	}

	return nil
}

// ImmediateSchedule 加入cron任务后立即运行
// https://github.com/robfig/cron
type ImmediateSchedule struct {
	first    int32
	schedule cron.Schedule
}

func NewImmediateSchedule(spec string) (*ImmediateSchedule, error) {
	parser := cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)
	schedule, err := parser.Parse(spec)
	if err != nil {
		return nil, err
	}

	return &ImmediateSchedule{schedule: schedule}, nil
}

func (schedule *ImmediateSchedule) Next(t time.Time) time.Time {
	if schedule.first == 0 {
		schedule.first = 1
		return t
	}

	return schedule.schedule.Next(t)
}
