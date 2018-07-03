package profitbricks

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/profitbricks/profitbricks-sdk-go"
)

func resourceProfitBricksGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceProfitBricksGroupCreate,
		Read:   resourceProfitBricksGroupRead,
		Update: resourceProfitBricksGroupUpdate,
		Delete: resourceProfitBricksGroupDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"create_datacenter": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"create_snapshot": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"reserve_ip": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"access_activity_log": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"user_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"users": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"first_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"last_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"email": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"password": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"administrator": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"force_sec_auth": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
		},

		Timeouts: &resourceDefaultTimeouts,
	}
}

func resourceProfitBricksGroupCreate(d *schema.ResourceData, meta interface{}) error {
	connection := meta.(*profitbricks.Client)
	request := profitbricks.Group{
		Properties: profitbricks.GroupProperties{},
	}

	log.Printf("[DEBUG] NAME %s", d.Get("name"))
	if d.Get("name") != nil {
		request.Properties.Name = d.Get("name").(string)
	}

	tempCreateDataCenter := d.Get("create_datacenter").(bool)
	request.Properties.CreateDataCenter = &tempCreateDataCenter
	tempCreateSnapshot := d.Get("create_snapshot").(bool)
	request.Properties.CreateSnapshot = &tempCreateSnapshot
	tempReserveIp := d.Get("reserve_ip").(bool)
	request.Properties.ReserveIP = &tempReserveIp
	tempAccessActivityLog := d.Get("access_activity_log").(bool)
	request.Properties.AccessActivityLog = &tempAccessActivityLog

	usertoAdd := d.Get("user_id").(string)

	group, err := connection.CreateGroup(request)

	log.Printf("[DEBUG] GROUP ID: %s", group.ID)

	if err != nil {
		return fmt.Errorf("An error occured while creating a group: %s", err)
	}

	// Wait, catching any errors
	_, errState := getStateChangeConf(meta, d, group.Headers.Get("Location"), schema.TimeoutCreate).WaitForState()
	if errState != nil {
		return errState
	}

	d.SetId(group.ID)

	//add users to group if any is provided
	if usertoAdd != "" {
		addedUser, err := connection.AddUserToGroup(d.Id(), usertoAdd)
		if err != nil {
			return fmt.Errorf("An error occured while adding %s user to group ID %s %s", usertoAdd, d.Id(), err)
		}
		// Wait, catching any errors
		_, errState := getStateChangeConf(meta, d, addedUser.Headers.Get("Location"), schema.TimeoutCreate).WaitForState()
		if errState != nil {
			return errState
		}
	}
	return resourceProfitBricksGroupRead(d, meta)
}

func resourceProfitBricksGroupRead(d *schema.ResourceData, meta interface{}) error {
	connection := meta.(*profitbricks.Client)
	group, err := connection.GetGroup(d.Id())

	if err != nil {
		if err2, ok := err.(profitbricks.ApiError); ok {
			if err2.HttpStatusCode() == 404 {
				d.SetId("")
				return nil
			}
		}
		return fmt.Errorf("An error occured while fetching a Group ID %s %s", d.Id(), err)
	}

	d.Set("name", group.Properties.Name)
	d.Set("create_datacenter", group.Properties.CreateDataCenter)
	d.Set("create_snapshot", group.Properties.CreateSnapshot)
	d.Set("reserve_ip", group.Properties.ReserveIP)
	d.Set("access_activity_log", group.Properties.AccessActivityLog)

	users, err := connection.ListGroupUsers(d.Id())
	if err != nil {
		return fmt.Errorf("An error occured while ListGroupUsers %s %s", d.Id(), err)
	}

	var usersArray = []profitbricks.UserProperties{}
	if len(users.Items) > 0 {
		for _, usr := range users.Items {
			usersArray = append(usersArray, *usr.Properties)
		}
		d.Set("users", usersArray)
	}

	return nil
}

func resourceProfitBricksGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	connection := meta.(*profitbricks.Client)
	tempCreateDataCenter := d.Get("create_datacenter").(bool)
	tempCreateSnapshot := d.Get("create_snapshot").(bool)
	tempReserveIp := d.Get("reserve_ip").(bool)
	tempAccessActivityLog := d.Get("access_activity_log").(bool)
	usertoAdd := d.Get("user_id").(string)
	groupReq := profitbricks.Group{
		Properties: profitbricks.GroupProperties{
			CreateDataCenter:  &tempCreateDataCenter,
			CreateSnapshot:    &tempCreateSnapshot,
			ReserveIP:         &tempReserveIp,
			AccessActivityLog: &tempAccessActivityLog,
		},
	}

	_, newValue := d.GetChange("name")
	groupReq.Properties.Name = newValue.(string)

	group, err := connection.UpdateGroup(d.Id(), groupReq)
	if err != nil {
		return fmt.Errorf("An error occured while patching a group ID %s %s", d.Id(), err)
	}
	// Wait, catching any errors
	_, errState := getStateChangeConf(meta, d, group.Headers.Get("Location"), schema.TimeoutUpdate).WaitForState()
	if errState != nil {
		return errState
	}

	//add users to group if any is provided
	if usertoAdd != "" {
		addedUser, err := connection.AddUserToGroup(d.Id(), usertoAdd)
		if err != nil {
			return fmt.Errorf("An error occured while adding %s user to group ID %s %s", usertoAdd, d.Id(), err)
		}

		// Wait, catching any errors
		_, errState := getStateChangeConf(meta, d, addedUser.Headers.Get("Location"), schema.TimeoutCreate).WaitForState()
		if errState != nil {
			return errState
		}
	}
	return resourceProfitBricksGroupRead(d, meta)
}

func resourceProfitBricksGroupDelete(d *schema.ResourceData, meta interface{}) error {
	connection := meta.(*profitbricks.Client)
	resp, err := connection.DeleteGroup(d.Id())
	if err != nil {
		//try again in 20 seconds
		time.Sleep(20 * time.Second)
		resp, err = connection.DeleteGroup(d.Id())

		if err != nil {
			if err2, ok := err.(profitbricks.ApiError); ok {
				if err2.HttpStatusCode() != 404 {
					return fmt.Errorf("An error occured while deleting a group %s %s", d.Id(), err)
				}
			}
		}
	}

	// Wait, catching any errors
	_, errState := getStateChangeConf(meta, d, resp.Get("Location"), schema.TimeoutDelete).WaitForState()
	if errState != nil {
		return errState
	}

	d.SetId("")
	return nil
}
