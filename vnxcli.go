package storages

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
)

type NavisecMessage struct {
	XMLName    xml.Name `xml:"CIM"`
	Text       string   `xml:",chardata"`
	CIMVERSION string   `xml:"CIMVERSION,attr"`
	DTDVERSION string   `xml:"DTDVERSION,attr"`
	MESSAGE    struct {
		Text            string `xml:",chardata"`
		ID              string `xml:"ID,attr"`
		PROTOCOLVERSION string `xml:"PROTOCOLVERSION,attr"`
		SIMPLERSP       struct {
			Text           string `xml:",chardata"`
			METHODRESPONSE struct {
				Text       string `xml:",chardata"`
				NAME       string `xml:"NAME,attr"`
				PARAMVALUE struct {
					Text  string `xml:",chardata"`
					NAME  string `xml:"NAME,attr"`
					TYPE  string `xml:"TYPE,attr"`
					VALUE struct {
						Text       string `xml:",chardata"`
						PARAMVALUE []struct {
							Text  string `xml:",chardata"`
							NAME  string `xml:"NAME,attr"`
							TYPE  string `xml:"TYPE,attr"`
							VALUE string `xml:"VALUE"`
						} `xml:"PARAMVALUE"`
					} `xml:"VALUE"`
				} `xml:"PARAMVALUE"`
				RETURNVALUE struct {
					Text               string `xml:",chardata"`
					TYPE               string `xml:"TYPE,attr"`
					VALUENAMEDINSTANCE struct {
						Text         string `xml:",chardata"`
						INSTANCENAME struct {
							Text      string `xml:",chardata"`
							CLASSNAME string `xml:"CLASSNAME,attr"`
						} `xml:"INSTANCENAME"`
						INSTANCE struct {
							Text      string `xml:",chardata"`
							CLASSNAME string `xml:"CLASSNAME,attr"`
							PROPERTY  []struct {
								Text  string `xml:",chardata"`
								NAME  string `xml:"NAME,attr"`
								TYPE  string `xml:"TYPE,attr"`
								VALUE string `xml:"VALUE"`
							} `xml:"PROPERTY"`
						} `xml:"INSTANCE"`
					} `xml:"VALUE.NAMEDINSTANCE"`
				} `xml:"RETURNVALUE"`
			} `xml:"METHODRESPONSE"`
		} `xml:"SIMPLERSP"`
	} `xml:"MESSAGE"`
}

type VNXLun struct {
	Id                    int64   //номер LUN
	Name                  string  //Имя LUN
	UID                   string  //UID уникальный в рамках одной СХД
	UserCapacityBlock     uint64  //
	UserCapacityGBs       float64 //
	ConsumedCapacityBlock uint64  //
	ConsumedCapacityGBs   float64 //
	PoolName              string  //
	RaidType              string  //
	IsPoolLUN             bool    //
	IsThinLUN             bool    //
	IsPrivate             bool    //
	IsCompressed          bool    //
}

type VNXHBAPort struct {
	SPName           string //
	SPPortID         uint64 //
	HBADeviceName    string //
	Trusted          bool   //
	LoggedIn         bool   //
	SourceID         uint64 //
	Defined          bool   //
	InitiatorType    int    //
	StorageGroupName string //
}

type VNXHBA struct {
	Id                   int64         //Id в справочнике
	HBAUID               string        //
	ServerName           string        //
	ServerIPAdress       string        //
	HBAModelDescription  string        //
	HBAVendorDescription string        //
	HBADEviceDriverName  string        //
	PortList             []*VNXHBAPort //
}

type VNXDisk struct {
	Id                    int64  //Id в справочнике
	VendorId              string //Вендор
	ProductId             string //идентификатор продукта
	ProductRevision       string //
	ClariionPartNumber    string //номер продукта
	ClariionTLAPartNumber string //
	DriveType             string //тип диска SAS ...
	SerialNumber          string //серийный номер
	Capacity              int64  //емкость
	RaidGroupId           string //
	ActualCapacity        int64  //
	LBAofUserSpace        int64  //
	Bus                   uint32 //шина
	Enclosure             uint32 //полка
	Disk                  uint32 //диск
}

func GetHBAInfoCmd(adress string) ([]*VNXHBA, error) {
	var msg NavisecMessage
	var hba []*VNXHBA
	var errorCode uint64
	var success bool

	cmd := exec.Command("/opt/Navisphere/bin/naviseccli", "-address", adress, "-secfilepath", "/var/local/loadmon", "-xml", "port", "-list", "-hba")
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		log.Println("Ошибка запуска процесса: ", err)
		return nil, errors.New("Error start command")
	}

	err = xml.Unmarshal(out.Bytes(), &msg)
	if err != nil {
		fmt.Println("Ошибка преобразования XML:", err)
		return nil, errors.New("Error unmarshal XML")
	}

	for _, stat := range msg.MESSAGE.SIMPLERSP.METHODRESPONSE.RETURNVALUE.VALUENAMEDINSTANCE.INSTANCE.PROPERTY {
		switch {
		case stat.NAME == "errorCode" && stat.TYPE == "uint32":
			errorCode, _ = strconv.ParseUint(stat.VALUE, 10, 32)
		case stat.NAME == "success" && stat.TYPE == "boolean":
			success, _ = strconv.ParseBool(stat.VALUE)
		}
	}

	if errorCode != 0 || !success {
		fmt.Println("Ошибка в сообщении NaviSecCLI:", errorCode, success)
		return nil, errors.New("Error return message NaviSecCLI")
	}
	i := 0
	j := 0
	for _, val := range msg.MESSAGE.SIMPLERSP.METHODRESPONSE.PARAMVALUE.VALUE.PARAMVALUE {

		fields := strings.Fields(val.NAME)
		if len(fields) == 0 { //нет полей пустая строка
			continue
		}
		switch {
		case val.NAME == "HBA UID" &&
			val.TYPE == "string":
			hba = append(hba, new(VNXHBA))
			hba[i].HBAUID = strings.TrimRight(val.VALUE, " ")
			i++
		case val.NAME == "Server Name" &&
			val.TYPE == "string":
			hba[i-1].ServerName = strings.TrimRight(val.VALUE, " ")
		case val.NAME == "Server IP Address" &&
			val.TYPE == "string":
			hba[i-1].ServerIPAdress = strings.TrimRight(val.VALUE, " ")
		case val.NAME == "HBA Model Description" &&
			val.TYPE == "string":
			hba[i-1].HBAModelDescription = strings.TrimRight(val.VALUE, " ")
		case val.NAME == "HBA Vendor Description" &&
			val.TYPE == "string":
			hba[i-1].HBAVendorDescription = strings.TrimRight(val.VALUE, " ")
		case val.NAME == "HBA Device Driver Name" &&
			val.TYPE == "string":
			hba[i-1].HBADEviceDriverName = strings.TrimRight(val.VALUE, " ")
		case val.NAME == "Information about each port of this HBA" &&
			val.TYPE == "string":
			j = 0
		case val.NAME == "    SP Name" &&
			val.TYPE == "string":
			hba[i-1].PortList = append(hba[i-1].PortList, new(VNXHBAPort))
			j++
			hba[i-1].PortList[j-1].SPName = strings.TrimRight(val.VALUE, " ")
		case val.NAME == "    SP Port ID" &&
			val.TYPE == "uint64":
			tmp, _ := strconv.ParseUint(val.VALUE, 10, 64)
			hba[i-1].PortList[j-1].SPPortID = tmp

		case val.NAME == "    HBA Devicename" &&
			val.TYPE == "string":
			hba[i-1].PortList[j-1].HBADeviceName = strings.TrimRight(val.VALUE, " ")

		case val.NAME == "    Trusted" &&
			val.TYPE == "string":
			if val.VALUE == "YES" {
				hba[i-1].PortList[j-1].Trusted = true
			} else {
				hba[i-1].PortList[j-1].Trusted = false
			}
		case val.NAME == "    Logged In" &&
			val.TYPE == "string":
			if val.VALUE == "YES" {
				hba[i-1].PortList[j-1].LoggedIn = true
			} else {
				hba[i-1].PortList[j-1].LoggedIn = false
			}
		case val.NAME == "    Defined" &&
			val.TYPE == "string":
			if val.VALUE == "YES" {
				hba[i-1].PortList[j-1].Defined = true
			} else {
				hba[i-1].PortList[j-1].Defined = false
			}
		case val.NAME == "    Source ID" &&
			val.TYPE == "string":
			tmp, _ := strconv.ParseUint(val.VALUE, 10, 64)
			hba[i-1].PortList[j-1].SourceID = tmp
		case val.NAME == "    Initiator Type" &&
			val.TYPE == "string":
			tmp, _ := strconv.ParseUint(val.VALUE, 10, 32)
			hba[i-1].PortList[j-1].InitiatorType = int(tmp)
		case val.NAME == "    StorageGroup Name" &&
			val.TYPE == "string":
			hba[i-1].PortList[j-1].StorageGroupName = strings.TrimRight(val.VALUE, " ")
		}
	}
	return hba, nil
}

func GetDisksCmd(adress string) ([]*VNXDisk, error) {
	var msg NavisecMessage
	var disks []*VNXDisk
	var errorCode uint64
	var success bool
	cmd := exec.Command("/opt/Navisphere/bin/naviseccli", "-address", adress, "-secfilepath", "/var/local/loadmon", "-xml", "getdisk", "-all")
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		log.Println("Ошибка запуска процесса: ", err)
		return nil, errors.New("Error start command")
	}

	err = xml.Unmarshal(out.Bytes(), &msg)
	if err != nil {
		fmt.Println("Ошибка преобразования XML:", err)
		return nil, errors.New("Error unmarshal XML")
	}

	for _, stat := range msg.MESSAGE.SIMPLERSP.METHODRESPONSE.RETURNVALUE.VALUENAMEDINSTANCE.INSTANCE.PROPERTY {
		switch {
		case stat.NAME == "errorCode" && stat.TYPE == "uint32":
			errorCode, _ = strconv.ParseUint(stat.VALUE, 10, 32)
		case stat.NAME == "success" && stat.TYPE == "boolean":
			success, _ = strconv.ParseBool(stat.VALUE)
		}
	}

	if errorCode != 0 || !success {
		fmt.Println("Ошибка в сообщении NaviSecCLI:", errorCode, success)
		return nil, errors.New("Error return message NaviSecCLI")
	}
	i := 0
	for _, val := range msg.MESSAGE.SIMPLERSP.METHODRESPONSE.PARAMVALUE.VALUE.PARAMVALUE {

		fields := strings.Fields(val.NAME)
		if len(fields) == 0 { //нет полей пустая строка
			continue
		}
		switch {
		case fields[0] == "Bus" &&
			fields[2] == "Enclosure" &&
			fields[4] == "Disk" &&
			val.TYPE == "string":
			disks = append(disks, new(VNXDisk))
			tmp, _ := strconv.ParseUint(fields[1], 10, 32)
			disks[i].Bus = uint32(tmp)
			tmp, _ = strconv.ParseUint(fields[3], 10, 32)
			disks[i].Enclosure = uint32(tmp)
			tmp, _ = strconv.ParseUint(fields[5], 10, 32)
			disks[i].Disk = uint32(tmp)
			i++
		case val.NAME == "Vendor Id" &&
			val.TYPE == "string":
			disks[i-1].VendorId = strings.TrimRight(val.VALUE, " ")
		case val.NAME == "Product Id" &&
			val.TYPE == "string":
			disks[i-1].ProductId = strings.TrimRight(val.VALUE, " ")
		case val.NAME == "Product Revision" &&
			val.TYPE == "string":
			disks[i-1].ProductRevision = strings.TrimRight(val.VALUE, " ")
		case val.NAME == "Serial Number" &&
			val.TYPE == "string":
			disks[i-1].SerialNumber = strings.TrimRight(val.VALUE, " ")
		case val.NAME == "Capacity" &&
			val.TYPE == "uint64":
			tmp, _ := strconv.ParseInt(val.VALUE, 10, 64)
			disks[i-1].Capacity = tmp

		case val.NAME == "Clariion Part Number" &&
			val.TYPE == "string":
			disks[i-1].ClariionPartNumber = strings.TrimRight(val.VALUE, " ")

		case val.NAME == "Raid Group ID" &&
			val.TYPE == "string":
			disks[i-1].RaidGroupId = strings.TrimRight(val.VALUE, " ")
		case val.NAME == "Drive Type" &&
			val.TYPE == "string":
			disks[i-1].DriveType = val.VALUE
		case val.NAME == "Clariion TLA Part Number" &&
			val.TYPE == "string":
			disks[i-1].ClariionTLAPartNumber = strings.TrimRight(val.VALUE, " ")
		case val.NAME == "Actual Capacity" &&
			val.TYPE == "string":
			tmp, _ := strconv.ParseInt(val.VALUE, 10, 64)
			disks[i-1].ActualCapacity = tmp
		case val.NAME == "LBA of User Space" &&
			val.TYPE == "string":
			tmp, _ := strconv.ParseInt(val.VALUE, 10, 64)
			disks[i-1].LBAofUserSpace = tmp
		}
	}
	return disks, nil
}

func GetLunsCmd(adress string) ([]*VNXLun, error) {
	var msg NavisecMessage
	var luns []*VNXLun
	var errorCode uint64
	var success bool

	cmd := exec.Command("/opt/Navisphere/bin/naviseccli", "-address", adress, "-secfilepath", "/var/local/loadmon", "-xml", "lun", "-list")
	//	cmd := exec.Command("/opt/Navisphere/bin/naviseccli","-user","vnxview","-password","Dy[1kerc", "-scope", "0", "-address", address, "-xml","lun", "-list")
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		log.Println("Ошибка запуска процесса: ", err, cmd.String())
		return nil, errors.New("Error start command")
	}

	err = xml.Unmarshal(out.Bytes(), &msg)
	if err != nil {
		fmt.Println("Ошибка преобразования XML:", err)
		return nil, errors.New("Error unmarshal XML")
	}

	for _, stat := range msg.MESSAGE.SIMPLERSP.METHODRESPONSE.RETURNVALUE.VALUENAMEDINSTANCE.INSTANCE.PROPERTY {
		switch {
		case stat.NAME == "errorCode" && stat.TYPE == "uint32":
			errorCode, _ = strconv.ParseUint(stat.VALUE, 10, 32)
		case stat.NAME == "success" && stat.TYPE == "boolean":
			success, _ = strconv.ParseBool(stat.VALUE)
		}
	}

	if errorCode != 0 || !success {
		fmt.Println("Ошибка в сообщении NaviSecCLI:", errorCode, success)
		return nil, errors.New("Error return message NaviSecCLI")
	}

	i := 0
	for _, val := range msg.MESSAGE.SIMPLERSP.METHODRESPONSE.PARAMVALUE.VALUE.PARAMVALUE {
		switch {
		case strings.TrimRight(val.NAME, " ") == "LOGICAL UNIT NUMBER" &&
			val.TYPE == "uint64":
			luns = append(luns, new(VNXLun))
			i++
			luns[i-1].Id, _ = strconv.ParseInt(val.VALUE, 10, 64)
		case val.NAME == "Name" &&
			val.TYPE == "string":
			luns[i-1].Name = val.VALUE
		case val.NAME == "UID" &&
			val.TYPE == "string":
			luns[i-1].UID = val.VALUE
		case val.NAME == "User Capacity (GBs)" &&
			val.TYPE == "string":
			luns[i-1].UserCapacityGBs, _ = strconv.ParseFloat(val.VALUE, 64)
		case val.NAME == "User Capacity (Blocks)" &&
			val.TYPE == "uint64":
			luns[i-1].UserCapacityBlock, _ = strconv.ParseUint(val.VALUE, 10, 64)
		case val.NAME == "Consumed Capacity (GBs)" &&
			val.TYPE == "string":
			luns[i-1].ConsumedCapacityGBs, _ = strconv.ParseFloat(val.VALUE, 64)
		case val.NAME == "Consumed Capacity (Blocks)" &&
			val.TYPE == "uint64":
			luns[i-1].ConsumedCapacityBlock, _ = strconv.ParseUint(val.VALUE, 10, 64)
		case val.NAME == "Pool Name" &&
			val.TYPE == "string":
			luns[i-1].PoolName = val.VALUE
		case val.NAME == "Raid Type" &&
			val.TYPE == "string":
			luns[i-1].RaidType = val.VALUE
		case val.NAME == "Is Pool LUN" &&
			val.TYPE == "string":
			if val.VALUE == "Yes" {
				luns[i-1].IsPoolLUN = true
			}
		case val.NAME == "Is Thin LUN" &&
			val.TYPE == "string":
			if val.VALUE == "Yes" {
				luns[i-1].IsThinLUN = true
			}
		case val.NAME == "Is Compressed" &&
			val.TYPE == "string":
			if val.VALUE == "Yes" {
				luns[i-1].IsCompressed = true
			}
		}
	}
	return luns, nil
}

/*
type ConnectVNX struct {
	EndPoint  string     //
	LunsFlag  chan bool  //
	DisksFlag chan bool  //
	PortsFlag chan bool  //
	Luns      []*VNXLun  //
	Disks     []*VNXDisk //
	Ports     []*VNXHBA  //
}

func main() {
	list := []ConnectVNX{
		{EndPoint: "10.2.19.112", LunsFlag: make(chan bool), DisksFlag: make(chan bool), PortsFlag: make(chan bool)},
		{EndPoint: "10.2.19.114", LunsFlag: make(chan bool), DisksFlag: make(chan bool), PortsFlag: make(chan bool)},
		{EndPoint: "10.2.19.116", LunsFlag: make(chan bool), DisksFlag: make(chan bool), PortsFlag: make(chan bool)},
		{EndPoint: "10.2.19.120", LunsFlag: make(chan bool), DisksFlag: make(chan bool), PortsFlag: make(chan bool)},
		{EndPoint: "10.2.19.122", LunsFlag: make(chan bool), DisksFlag: make(chan bool), PortsFlag: make(chan bool)},
		{EndPoint: "10.2.19.124", LunsFlag: make(chan bool), DisksFlag: make(chan bool), PortsFlag: make(chan bool)},
		{EndPoint: "10.2.19.126", LunsFlag: make(chan bool), DisksFlag: make(chan bool), PortsFlag: make(chan bool)},
		{EndPoint: "10.2.19.130", LunsFlag: make(chan bool), DisksFlag: make(chan bool), PortsFlag: make(chan bool)},
	}

	for i := range list {
		fmt.Println(i, "start request", list[i].EndPoint)
		go func() {
			list[i].Luns, _ = getLunsCmd(list[i].EndPoint)
			list[i].LunsFlag <- true
		}()
		go func() {
			list[i].Disks, _ = getDisksCmd(list[i].EndPoint)
			list[i].DisksFlag <- true
		}()
		go func() {
			list[i].Ports, _ = getHBAInfoCmd(list[i].EndPoint)
			list[i].PortsFlag <- true
		}()
		<-list[i].LunsFlag
		<-list[i].DisksFlag
		<-list[i].PortsFlag
	}
	for j := range list {
		fmt.Println(j, "result", list[j].EndPoint)
		//	    <-list[j].LunsFlag
		for i, luns := range list[j].Luns {
			fmt.Println(i, luns.Id, luns.Name, luns.UserCapacityGBs)
		}

		//	    <-list[j].DisksFlag
		for i, disks := range list[j].Disks {
			fmt.Println(i, "Bus", disks.Bus, "Enclosure", disks.Enclosure, "Slot", disks.Disk, disks.Id, disks.VendorId, disks.ProductId, disks.SerialNumber)
		}

		//	    <-list[j].PortsFlag
		for i, ports := range list[j].Ports {
			fmt.Println(i, ports.Id, ports.HBAUID, ports.ServerName, ports.ServerIPAdress)
		}
	}
}
*/
