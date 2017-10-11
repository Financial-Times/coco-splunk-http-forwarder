package main

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

const event = `{"@time":"2017-08-18T14:37:15.639583741Z","HOSTNAME":"test_host","MACHINE_ID":"machine_id","MESSAGE":"{"@time":"2017-08-18T14:37:15.639583741Z","content_type":"Annotations","event":"mapping","isValid":"true","level":"info","monitoring_event":"true","msg":"Successfully mapped","service_name":"annotations-mapper","transaction_id":"tid_rahiuyzv8d","uuid":"a64cdd19-7cfe-1147-ab12-a13271d1dd9c"}","SYSTEMD_UNIT":"annotations-mapper@2.service","_SYSTEMD_INVOCATION_ID":"512d67a816cc44ceb6d0c1e8bd3702f9","content_type":"Annotations","event":"mapping","isValid":"true","level":"info","monitoring_event":"true","msg":"Successfully mapped","platform":"up-coco","service_name":"annotations-mapper","transaction_id":"tid_rahiuyzv8d","uuid":"a64cdd19-7cfe-1147-ab12-a13271d1dd9c"}`

func Test_WriteJson_Success(t *testing.T) {
	eventList := []string{event}
	expected := ` {"event":"{\"@time\":\"2017-08-18T14:37:15.639583741Z\",\"HOSTNAME\":\"test_host\",\"MACHINE_ID\":\"machine_id\",\"MESSAGE\":\"{\"@time\":\"2017-08-18T14:37:15.639583741Z\",\"content_type\":\"Annotations\",\"event\":\"mapping\",\"isValid\":\"true\",\"level\":\"info\",\"monitoring_event\":\"true\",\"msg\":\"Successfully mapped\",\"service_name\":\"annotations-mapper\",\"transaction_id\":\"tid_rahiuyzv8d\",\"uuid\":\"a64cdd19-7cfe-1147-ab12-a13271d1dd9c\"}\",\"SYSTEMD_UNIT\":\"annotations-mapper@2.service\",\"_SYSTEMD_INVOCATION_ID\":\"512d67a816cc44ceb6d0c1e8bd3702f9\",\"content_type\":\"Annotations\",\"event\":\"mapping\",\"isValid\":\"true\",\"level\":\"info\",\"monitoring_event\":\"true\",\"msg\":\"Successfully mapped\",\"platform\":\"up-coco\",\"service_name\":\"annotations-mapper\",\"transaction_id\":\"tid_rahiuyzv8d\",\"uuid\":\"a64cdd19-7cfe-1147-ab12-a13271d1dd9c\"}","time":1503067035.639}`
	actual := writeJSON(eventList)
	assert.Equal(t, expected, actual)
}

func Test_WriteJson_MissingTimestamp(t *testing.T) {
	event := `{"HOSTNAME":"test_host","MACHINE_ID":"machine_id","MESSAGE":"{\"content_type\":\"Annotations\",\"event\":\"mapping\",\"isValid\":\"true\",\"level\":\"info\",\"monitoring_event\":\"true\",\"msg\":\"Successfully mapped\",\"service_name\":\"annotations-mapper\",\"transaction_id\":\"tid_rahiuyzv8d\",\"uuid\":\"a64cdd19-7cfe-1147-ab12-a13271d1dd9c\"}","SYSTEMD_UNIT":"annotations-mapper@2.service","_SYSTEMD_INVOCATION_ID":"512d67a816cc44ceb6d0c1e8bd3702f9","content_type":"Annotations","event":"mapping","isValid":"true","level":"info","monitoring_event":"true","msg":"Successfully mapped","platform":"up-coco","service_name":"annotations-mapper","transaction_id":"tid_rahiuyzv8d","uuid":"a64cdd19-7cfe-1147-ab12-a13271d1dd9c"}`
	eventList := []string{event}
	jsonResult := map[string]interface{}{}
	err := json.Unmarshal([]byte(writeJSON(eventList)), &jsonResult)
	assert.NoError(t, err)
	_, found := jsonResult["time"].(float64)
	assert.True(t, found)
}

func Test_Determine_Index(t *testing.T) {

	var tests = []struct {
		event string
		index string
	}{
		{event:`"HOSTNAME":"test_host","environment":"pub-xp","monitoring_event":"true","MACHINE_ID":"machine_id","MESSAGE":"{\"content_type\":\"Annotations\",\"event\":\"mapping\",\"isValid\":\"true\",\"level\":\"info\",\"monitoring_event\":\"true\",\"msg\":\"Successfully mapped\",\"service_name\":\"annotations-mapper\",\"transaction_id\":\"tid_rahiuyzv8d\",\"uuid\":\"a64cdd19-7cfe-1147-ab12-a13271d1dd9c\"}","SYSTEMD_UNIT":"annotations-mapper@2.service","_SYSTEMD_INVOCATION_ID":"512d67a816cc44ceb6d0c1e8bd3702f9","content_type":"Annotations","event":"mapping","isValid":"true","level":"info","monitoring_event":"true","msg":"Successfully mapped","platform":"up-coco","service_name":"annotations-mapper","transaction_id":"tid_rahiuyzv8d","uuid":"a64cdd19-7cfe-1147-ab12-a13271d1dd9c"}`,
			index:"xp_mon",
		},
		{event:`"HOSTNAME":"test_host","environment":"xp","monitoring_event":"true","MACHINE_ID":"machine_id","MESSAGE":"{\"content_type\":\"Annotations\",\"event\":\"mapping\",\"isValid\":\"true\",\"level\":\"info\",\"monitoring_event\":\"true\",\"msg\":\"Successfully mapped\",\"service_name\":\"annotations-mapper\",\"transaction_id\":\"tid_rahiuyzv8d\",\"uuid\":\"a64cdd19-7cfe-1147-ab12-a13271d1dd9c\"}","SYSTEMD_UNIT":"annotations-mapper@2.service","_SYSTEMD_INVOCATION_ID":"512d67a816cc44ceb6d0c1e8bd3702f9","content_type":"Annotations","event":"mapping","isValid":"true","level":"info","monitoring_event":"true","msg":"Successfully mapped","platform":"up-coco","service_name":"annotations-mapper","transaction_id":"tid_rahiuyzv8d","uuid":"a64cdd19-7cfe-1147-ab12-a13271d1dd9c"}`,
			index:"xp_mon",
		},
		{event:`"HOSTNAME":"test_host","environment":"xp-data","monitoring_event":"true","MACHINE_ID":"machine_id","MESSAGE":"{\"content_type\":\"Annotations\",\"event\":\"mapping\",\"isValid\":\"true\",\"level\":\"info\",\"monitoring_event\":\"true\",\"msg\":\"Successfully mapped\",\"service_name\":\"annotations-mapper\",\"transaction_id\":\"tid_rahiuyzv8d\",\"uuid\":\"a64cdd19-7cfe-1147-ab12-a13271d1dd9c\"}","SYSTEMD_UNIT":"annotations-mapper@2.service","_SYSTEMD_INVOCATION_ID":"512d67a816cc44ceb6d0c1e8bd3702f9","content_type":"Annotations","event":"mapping","isValid":"true","level":"info","monitoring_event":"true","msg":"Successfully mapped","platform":"up-coco","service_name":"annotations-mapper","transaction_id":"tid_rahiuyzv8d","uuid":"a64cdd19-7cfe-1147-ab12-a13271d1dd9c"}`,
			index:"xp_mon",
		},
		{event:`"HOSTNAME":"test_host","environment":"pub-xp","MACHINE_ID":"machine_id","MESSAGE":"{\"content_type\":\"Annotations\",\"event\":\"mapping\",\"isValid\":\"true\",\"level\":\"info\",\"msg\":\"Successfully mapped\",\"service_name\":\"annotations-mapper\",\"transaction_id\":\"tid_rahiuyzv8d\",\"uuid\":\"a64cdd19-7cfe-1147-ab12-a13271d1dd9c\"}","SYSTEMD_UNIT":"annotations-mapper@2.service","_SYSTEMD_INVOCATION_ID":"512d67a816cc44ceb6d0c1e8bd3702f9","content_type":"Annotations","event":"mapping","isValid":"true","level":"info","msg":"Successfully mapped","platform":"up-coco","service_name":"annotations-mapper","transaction_id":"tid_rahiuyzv8d","uuid":"a64cdd19-7cfe-1147-ab12-a13271d1dd9c"}`,
			index:"xp",
		},
		{event:`"HOSTNAME":"test_host","environment":"xp","MACHINE_ID":"machine_id","MESSAGE":"{\"content_type\":\"Annotations\",\"event\":\"mapping\",\"isValid\":\"true\",\"level\":\"info\",\"msg\":\"Successfully mapped\",\"service_name\":\"annotations-mapper\",\"transaction_id\":\"tid_rahiuyzv8d\",\"uuid\":\"a64cdd19-7cfe-1147-ab12-a13271d1dd9c\"}","SYSTEMD_UNIT":"annotations-mapper@2.service","_SYSTEMD_INVOCATION_ID":"512d67a816cc44ceb6d0c1e8bd3702f9","content_type":"Annotations","event":"mapping","isValid":"true","level":"info","msg":"Successfully mapped","platform":"up-coco","service_name":"annotations-mapper","transaction_id":"tid_rahiuyzv8d","uuid":"a64cdd19-7cfe-1147-ab12-a13271d1dd9c"}`,
			index:"xp",
		},
		{event:`"HOSTNAME":"test_host","environment":"xp-data","MACHINE_ID":"machine_id","MESSAGE":"{\"content_type\":\"Annotations\",\"event\":\"mapping\",\"isValid\":\"true\",\"level\":\"info\",\"monitoring_event\":\"true\",\"msg\":\"Successfully mapped\",\"service_name\":\"annotations-mapper\",\"transaction_id\":\"tid_rahiuyzv8d\",\"uuid\":\"a64cdd19-7cfe-1147-ab12-a13271d1dd9c\"}","SYSTEMD_UNIT":"annotations-mapper@2.service","_SYSTEMD_INVOCATION_ID":"512d67a816cc44ceb6d0c1e8bd3702f9","content_type":"Annotations","event":"mapping","isValid":"true","level":"info","msg":"Successfully mapped","platform":"up-coco","service_name":"annotations-mapper","transaction_id":"tid_rahiuyzv8d","uuid":"a64cdd19-7cfe-1147-ab12-a13271d1dd9c"}`,
			index:"xp",
		},
	}

	for _, test := range tests {
		index := determineIndex(test.event)
		assert.Equal(t, test.index, index)
	}
}