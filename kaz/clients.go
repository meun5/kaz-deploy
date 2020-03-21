package kaz

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type Group struct {
	Name string
	Tags []string
}

type Client struct {
	VMWareName string
	VMWareId   string `gorm:"unique;not null"`
	MacAddress string `gorm:"unique;not null"`
	Team       int
	Group      Group
	CheckedIn  bool `gorm:"unique;not null;default:false"`
	gorm.Model
}

const DatabaseName string = "kaz.db"

const (
	TeamFoldersPrefix string = "CDC"
	TeamVMPrefix      string = "Team"
	RootFolder        string = "Competition Folder"
)

var (
	FoldersIds         = make(map[string]string)
	VirtualMachinesIds = make(map[string]string)
	//VirtualMachineNetworkIds  = make(map[string][]int)
	VirtualMachineNetworkMacs = make(map[string]string)

	sessionToken = getSessionToken()
)

func InitializeDatabase(s *Server) error {
	if s.Db != nil {
		return nil
	}

	edb, err := gorm.Open("sqlite3", DatabaseName)
	if err != nil {
		return err
	}

	s.Db = edb

	return nil
}

func getSessionToken() string {
	u := url.URL{
		Host:   os.Getenv("VCENTER_SERVER"),
		Scheme: "https",
		Path:   "/rest/com/vmware/cis/session",
	}

	client := http.DefaultClient
	r, err := http.NewRequest("POST", u.String(), nil)
	if err != nil {
		fmt.Printf("Error Making HTTP Request: %+v\n", err)
		return ""
	}

	r.SetBasicAuth(os.Getenv("VCENTER_USER"), os.Getenv("VCENTER_PASSWORD"))
	q, err := client.Do(r)
	if err != nil {
		fmt.Printf("Error Executing HTTP Request: %+v\n", err)
		return ""
	}

	b, err := ioutil.ReadAll(q.Body)
	if err != nil {
		fmt.Printf("Error Reading HTTP Response: %+v\n", err)
		return ""
	}

	var value struct {
		Value string `json:"value"`
	}

	fmt.Printf("SESSION FROM VCENTER: %s\n", string(b))

	err = json.Unmarshal(b, &value)
	if err != nil {
		fmt.Printf("Error Parsing HTTP Response: %+v\n", err)
		return ""
	}

	return value.Value
}

func Send(u url.URL, sessionToken string) ([]byte, error) {
	u.Host = os.Getenv("VCENTER_SERVER")
	u.Scheme = "https"

	fmt.Printf("URL: %s\n", u.String())

	r, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	r.Header.Add("vmware-api-session-id", sessionToken)

	o, err := http.DefaultClient.Do(r)
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(o.Body)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func InitializeCache(override bool) error {
	if len(FoldersIds) == 0 || override {
		if sessionToken == "" {
			sessionToken = getSessionToken()
		}

		params := url.Values{}
		params.Add("filter.type", "VIRTUAL_MACHINE")
		params.Add("filter.names", RootFolder)

		u := url.URL{}
		u.RawQuery = params.Encode()
		u.Path = "/rest/vcenter/folder"

		b, err := Send(u, sessionToken)
		if err != nil {
			fmt.Printf("Error With Executing API Request: %+v\n", err)
			return err
		}

		fmt.Printf("RESPONSE FROM VCENTER: %s\n", string(b))

		var folder struct {
			Value []struct {
				Folder string `json:"folder"`
				Name   string `json:"name"`
				Type   string `json:"type"`
			} `json:"value"`
		}

		err = json.Unmarshal(b, &folder)
		if err != nil {
			return err
		}

		fmt.Printf("Returned Data from vCenter: %+v\n", folder)

		for _, t := range folder.Value {
			fmt.Printf("Mapping Folder \"%s\" to %s\n", t.Name, t.Folder)
			FoldersIds[t.Name] = t.Folder
		}

		fmt.Printf("Resulting Folder Map: %+v\n", FoldersIds)

		// Get the subfolders. (i.e. do it again)
		params = url.Values{}
		params.Add("filter.type", "VIRTUAL_MACHINE")
		params.Add("filter.parent_folders", FoldersIds[RootFolder])

		u = url.URL{}
		u.RawQuery = params.Encode()
		u.Path = "/rest/vcenter/folder"

		b, err = Send(u, sessionToken)
		if err != nil {
			fmt.Printf("Error With Executing API Request: %+v\n", err)
			return err
		}

		fmt.Printf("RESPONSE FROM VCENTER: %s\n", string(b))

		err = json.Unmarshal(b, &folder)
		if err != nil {
			return err
		}

		fmt.Printf("Returned Data from vCenter: %+v\n", folder)

		for _, t := range folder.Value {
			if !strings.HasPrefix(t.Name, TeamFoldersPrefix) {
				fmt.Printf("Dropping Folder \"%s\" with id %s\n", t.Name, t.Folder)
				continue
			}

			fmt.Printf("Mapping Folder \"%s\" to %s\n", t.Name, t.Folder)
			FoldersIds[t.Name] = t.Folder
		}

		fmt.Printf("Resulting Folder Map: %+v\n", FoldersIds)
	}

	// Get vms in those folders
	if len(VirtualMachinesIds) == 0 || override {
		if sessionToken == "" {
			sessionToken = getSessionToken()
		}

		for _, j := range FoldersIds {
			if j == RootFolder {
				continue
			}

			params := url.Values{}
			params.Add("filter.folders", j)

			u := url.URL{}
			u.RawQuery = params.Encode()
			u.Path = "/rest/vcenter/vm"

			b, err := Send(u, sessionToken)
			if err != nil {
				fmt.Printf("Error With Executing API Request: %+v\n", err)
				return err
			}

			fmt.Printf("RESPONSE FROM VCENTER: %s\n", string(b))

			var vm struct {
				Value []struct {
					VM   string `json:"vm"`
					Name string `json:"name"`
				} `json:"value"`
			}

			err = json.Unmarshal(b, &vm)
			if err != nil {
				return err
			}

			fmt.Printf("Returned Data from vCenter: %+v\n", vm)
			for _, t := range vm.Value {
				if !strings.HasPrefix(t.Name, TeamVMPrefix) {
					fmt.Printf("Dropping VM \"%s\" with id %s\n", t.Name, t.VM)
					continue
				}

				fmt.Printf("Mapping VM \"%s\" to %s\n", t.Name, t.VM)
				VirtualMachinesIds[t.VM] = t.Name
			}

			fmt.Printf("Resulting VM Ids Map: %+v\n", VirtualMachinesIds)
		}
	}

	// Map Nic IDs to VMs
	/*if len(VirtualMachineNetworkIds) == 0 {
		if sessionToken == "" {
			sessionToken = getSessionToken()
		}

		for _, w := range VirtualMachinesIds {
			u := url.URL{}
			u.Path = fmt.Sprintf("/rest/vcenter/vm/%s/hardware/ethernet/", w)

			b, err := Send(u, sessionToken)
			if err != nil {
				fmt.Printf("Error With Executing API Request: %+v\n", err)
				return Client{}, nil
			}

			fmt.Printf("RESPONSE FROM VCENTER: %s\n", string(b))

			var nic struct {
				Value []struct {
					Nic   string `json:"nic"`
				} `json:"value"`
			}

			err = json.Unmarshal(b, &nic)
			if err != nil {
				return Client{}, err
			}

			fmt.Printf("Returned Data from vCenter: %+v\n", nic)
			for _, t := range nic.Value {
				r, err := strconv.Atoi(t.Nic)
				if err != nil {
					return Client{}, nil
				}

				fmt.Printf("Mapping VM NIC \"%d\" to %s\n", r, w)

				VirtualMachineNetworkIds[w] = append(VirtualMachineNetworkIds[w], r)
			}

			fmt.Printf("Resulting VM Network Ids Map: %+v\n", VirtualMachineNetworkIds)
		}
	}*/

	if len(VirtualMachineNetworkMacs) == 0 || override {
		if sessionToken == "" {
			sessionToken = getSessionToken()
		}

		for w := range VirtualMachinesIds {
			u := url.URL{}
			u.Path = fmt.Sprintf("/rest/vcenter/vm/%s/hardware/ethernet/4000", w)

			b, err := Send(u, sessionToken)
			if err != nil {
				fmt.Printf("Error With Executing API Request: %+v\n", err)
				return err
			}

			fmt.Printf("RESPONSE FROM VCENTER: %s\n", string(b))

			var nic struct {
				Value struct {
					MacAddress string `json:"mac_address"`
				} `json:"value"`
			}

			err = json.Unmarshal(b, &nic)
			if err != nil {
				return err
			}

			fmt.Printf("Returned Data from vCenter: %+v\n", nic)
			fmt.Printf("Mapping Mac Address \"%s\" to %s\n", nic.Value.MacAddress, w)
			VirtualMachineNetworkMacs[w] = nic.Value.MacAddress

			fmt.Printf("Resulting VM Network Ids Map: %+v\n", VirtualMachineNetworkMacs)
		}
	}

	return nil
}

func GetClientByMacAddress(address string) (*Client, error) {
	err := InitializeCache(false)
	if err != nil {
		return nil, err
	}

	var ke string
	for k, v := range VirtualMachineNetworkMacs {
		if v == address {
			fmt.Printf("VM Id for Mac Address \"%s\" is %s\n", address, k)
			ke = k
			break
		}
	}

	if ke == "" {
		return nil, fmt.Errorf("unable to find vm with that mac address")
	}

	fields := strings.Fields(VirtualMachinesIds[ke])

	if len(fields) < 3 {
		return nil, fmt.Errorf("dat vm name no have enough parts")
	}

	team, err := strconv.Atoi(fields[1])

	if err != nil {
		return nil, err
	}

	c := &Client{
		VMWareName: VirtualMachinesIds[ke],
		VMWareId:   ke,
		MacAddress: address,
		Team:       team,
		Group:      Group{},
		CheckedIn:  false,
	}

	return c, nil
}

func CommitClient(c *Client, s *Server) error {
	if s.Db == nil {
		return fmt.Errorf("uninitialized db")
	}

	s.Logger.Printf("Commiting new Client: (%t) %+v\n", s.Db.NewRecord(c), c)

	if s.Db.NewRecord(c) {
		s.Db.Create(c)
	}

	if s.Db.NewRecord(c) {
		return fmt.Errorf("unable to add client to database")
	}

	return nil
}
