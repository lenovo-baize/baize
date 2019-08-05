package gather

import (
	"errors"
)

var ReportDataChan chan map[string]string = make(chan map[string]string, 1)

const EventActionKey = "event_action"

var IsReportStarted = false

//Gather 收集数据
func Gather(data map[string]string) error {
	if nil == data {
		return nil
	}
	eventAction := data[EventActionKey]
	if "" == eventAction {
		return errors.New(EventActionKey + " cannot empty")
	}
	ReportDataChan <- data
	return nil
}
