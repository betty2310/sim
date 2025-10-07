package types

type ServerPushBiometricMessage struct {
	BioID    int    `json:"BioId"`
	IDNumber string `json:"IdNumber"`
	FullName string `json:"FullName"`
}

type DeviceSyncBioMessage struct {
	MacAddress string `json:"MacAddress"`
	LstBioId   []int  `json:"LstBioId"`
	DoorID     int    `json:"DoorId"`
}

type DeviceReceivedBioMessage struct {
	MacAddress string `json:"MacAddress"`
	BioID      int    `json:"BioId"`
	DeviceTime string `json:"DeviceTime"`
	CmdType    string `json:"CmdType"`
}
