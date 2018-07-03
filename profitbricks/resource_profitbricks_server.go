package profitbricks

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/profitbricks/profitbricks-sdk-go"
	"golang.org/x/crypto/ssh"
)

func resourceProfitBricksServer() *schema.Resource {
	return &schema.Resource{
		Create: resourceProfitBricksServerCreate,
		Read:   resourceProfitBricksServerRead,
		Update: resourceProfitBricksServerUpdate,
		Delete: resourceProfitBricksServerDelete,
		Schema: map[string]*schema.Schema{

			//Server parameters
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"cores": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"ram": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"availability_zone": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"licence_type": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"boot_volume": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"boot_cdrom": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cpu_family": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"boot_image": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"primary_nic": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"primary_ip": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"datacenter_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"volume": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"image_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"size": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"disk_type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"image_password": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"licence_type": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"ssh_key_path": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Optional: true,
						},
						"bus": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"name": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"availability_zone": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"nic": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"lan": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"name": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"dhcp": {
							Type:     schema.TypeBool,
							Optional: true,
						},

						"ip": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"ips": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Computed: true,
						},
						"nat": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"firewall_active": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"firewall": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Optional: true,
									},

									"protocol": {
										Type:     schema.TypeString,
										Required: true,
									},
									"source_mac": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"source_ip": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"target_ip": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"ip": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"ips": {
										Type:     schema.TypeList,
										Elem:     &schema.Schema{Type: schema.TypeString},
										Optional: true,
									},
									"port_range_start": {
										Type:     schema.TypeInt,
										Optional: true,
										ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
											if v.(int) < 1 && v.(int) > 65534 {
												errors = append(errors, fmt.Errorf("Port start range must be between 1 and 65534"))
											}
											return
										},
									},

									"port_range_end": {
										Type:     schema.TypeInt,
										Optional: true,
										ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
											if v.(int) < 1 && v.(int) > 65534 {
												errors = append(errors, fmt.Errorf("Port end range must be between 1 and 65534"))
											}
											return
										},
									},
									"icmp_type": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"icmp_code": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},
		},

		Timeouts: &resourceDefaultTimeouts,
	}
}

func resourceProfitBricksServerCreate(d *schema.ResourceData, meta interface{}) error {
	connection := meta.(*profitbricks.Client)

	var image_alias string
	request := profitbricks.Server{
		Properties: profitbricks.ServerProperties{
			Name:  d.Get("name").(string),
			Cores: d.Get("cores").(int),
			RAM:   d.Get("ram").(int),
		},
	}
	dcId := d.Get("datacenter_id").(string)

	isSnapshot := false
	if v, ok := d.GetOk("availability_zone"); ok {
		request.Properties.AvailabilityZone = v.(string)
	}

	if v, ok := d.GetOk("cpu_family"); ok {
		if v.(string) != "" {
			request.Properties.CPUFamily = v.(string)
		}
	}
	if vRaw, ok := d.GetOk("volume"); ok {

		volumeRaw := vRaw.(*schema.Set).List()

		for _, raw := range volumeRaw {
			rawMap := raw.(map[string]interface{})
			var imagePassword string
			//Can be one file or a list of files
			var sshkey_path []interface{}
			var image, licenceType, availabilityZone string

			if rawMap["image_password"] != nil {
				imagePassword = rawMap["image_password"].(string)
			}
			if rawMap["ssh_key_path"] != nil {
				sshkey_path = rawMap["ssh_key_path"].([]interface{})
			}

			image_name := rawMap["image_name"].(string)
			if !IsValidUUID(image_name) {
				image = getImageId(connection, dcId, image_name, rawMap["disk_type"].(string))
				//if no image id was found with that name we look for a matching snapshot
				if image == "" {
					image = getSnapshotId(connection, image_name)
					if image != "" {
						isSnapshot = true
					} else {
						dc, err := connection.GetDatacenter(dcId)
						if err != nil {
							return fmt.Errorf("Error fetching datacenter %s: (%s)", dcId, err)
						}
						image_alias = getImageAlias(connection, image_name, dc.Properties.Location)
					}
				}
				if image == "" && image_alias == "" {
					return fmt.Errorf("Could not find an image/imagealias/snapshot that matches %s ", image_name)
				}
				if imagePassword == "" && len(sshkey_path) == 0 && isSnapshot == false {
					return fmt.Errorf("Either 'image_password' or 'sshkey' must be provided.")
				}
			} else {
				img, err := connection.GetImage(image_name)

				var apiError profitbricks.ApiError

				if apiError, ok = err.(profitbricks.ApiError); !ok {
					return fmt.Errorf("Error fetching image %s: (%s)", image_name, err)
				}

				if apiError.HttpStatusCode() == 404 {
					img, err := connection.GetSnapshot(image_name)

					if apiError, ok := err.(profitbricks.ApiError); !ok {
						if apiError.HttpStatusCode() == 404 {
							return fmt.Errorf("image/snapshot: %s Not Found", img.Response)
						}
					}

					isSnapshot = true
				} else {
					if err != nil {
						return fmt.Errorf("Error fetching image/snapshot: %s", err)
					}
				}
				if img.Properties.Public == true && isSnapshot == false {
					if imagePassword == "" && len(sshkey_path) == 0 {
						return fmt.Errorf("Either 'image_password' or 'sshkey' must be provided.")
					}
					image = getImageId(connection, d.Get("datacenter_id").(string), image_name, rawMap["disk_type"].(string))
				} else {
					img, err := connection.GetImage(image_name)
					if err != nil {
						img, err := connection.GetSnapshot(image_name)
						if err != nil {
							return fmt.Errorf("Error fetching image/snapshot: %s", img.Response)
						}
						isSnapshot = true
					}
					if img.Properties.Public == true && isSnapshot == false {
						if imagePassword == "" && len(sshkey_path) == 0 {
							return fmt.Errorf("Either 'image_password' or 'sshkey' must be provided.")
						}
						image = image_name
					} else {
						image = image_name
					}
				}
			}

			if rawMap["licence_type"] != nil {
				licenceType = rawMap["licence_type"].(string)
			}

			var publicKeys []string
			if len(sshkey_path) != 0 {
				for _, path := range sshkey_path {
					log.Printf("[DEBUG] Reading file %s", path)
					publicKey, err := readPublicKey(path.(string))
					if err != nil {
						return fmt.Errorf("Error fetching sshkey from file (%s) %s", path, err.Error())
					}
					publicKeys = append(publicKeys, publicKey)
				}
			}
			if rawMap["availability_zone"] != nil {
				availabilityZone = rawMap["availability_zone"].(string)
			}
			if image == "" && licenceType == "" && image_alias == "" && !isSnapshot {
				return fmt.Errorf("Either 'image', 'licenceType', or 'imageAlias' must be set.")
			}

			if isSnapshot == true && (imagePassword != "" || len(publicKeys) > 0) {
				return fmt.Errorf("Passwords/SSH keys  are not supported for snapshots.")
			}

			request.Entities = &profitbricks.ServerEntities{
				Volumes: &profitbricks.Volumes{
					Items: []profitbricks.Volume{
						{
							Properties: profitbricks.VolumeProperties{
								Name:             rawMap["name"].(string),
								Size:             rawMap["size"].(int),
								Type:             rawMap["disk_type"].(string),
								ImagePassword:    imagePassword,
								Image:            image,
								ImageAlias:       image_alias,
								Bus:              rawMap["bus"].(string),
								LicenceType:      licenceType,
								AvailabilityZone: availabilityZone,
							},
						},
					},
				},
			}

			if len(publicKeys) == 0 {
				request.Entities.Volumes.Items[0].Properties.SSHKeys = nil
			} else {
				request.Entities.Volumes.Items[0].Properties.SSHKeys = publicKeys
			}
		}

	}

	if nRaw, ok := d.GetOk("nic"); ok {
		nicRaw := nRaw.(*schema.Set).List()

		for _, raw := range nicRaw {
			rawMap := raw.(map[string]interface{})
			nic := profitbricks.Nic{Properties: &profitbricks.NicProperties{}}
			if rawMap["lan"] != nil {
				nic.Properties.Lan = rawMap["lan"].(int)
			}
			if rawMap["name"] != nil {
				nic.Properties.Name = rawMap["name"].(string)
			}
			if rawMap["dhcp"] != nil {
				val := rawMap["dhcp"].(bool)
				nic.Properties.Dhcp = &val
			}
			if rawMap["firewall_active"] != nil {
				nic.Properties.FirewallActive = rawMap["firewall_active"].(bool)
			}
			if rawMap["ip"] != nil {
				rawIps := rawMap["ip"].(string)
				ips := strings.Split(rawIps, ",")
				if rawIps != "" {
					nic.Properties.Ips = ips
				}
			}
			if rawMap["nat"] != nil {
				nic.Properties.Nat = rawMap["nat"].(bool)
			}
			request.Entities.Nics = &profitbricks.Nics{
				Items: []profitbricks.Nic{
					nic,
				},
			}

			if rawMap["firewall"] != nil {
				rawFw := rawMap["firewall"].(*schema.Set).List()
				for _, rraw := range rawFw {
					fwRaw := rraw.(map[string]interface{})
					log.Println("[DEBUG] fwRaw", fwRaw["protocol"])

					firewall := profitbricks.FirewallRule{
						Properties: profitbricks.FirewallruleProperties{
							Protocol: fwRaw["protocol"].(string),
						},
					}

					if fwRaw["name"] != nil {
						firewall.Properties.Name = fwRaw["name"].(string)
					}
					if fwRaw["source_mac"] != "" {
						tempSourceMac := fwRaw["source_mac"].(string)
						firewall.Properties.SourceMac = &tempSourceMac
					}
					if fwRaw["source_ip"] != "" {
						tempSourceIp := fwRaw["source_ip"].(string)
						firewall.Properties.SourceIP = &tempSourceIp
					}
					if fwRaw["target_ip"] != "" {
						tempTargetIp := fwRaw["target_ip"].(string)
						firewall.Properties.TargetIP = &tempTargetIp
					}
					if fwRaw["port_range_start"] != nil {
						tempPortRangeStart := fwRaw["port_range_start"].(int)
						firewall.Properties.PortRangeStart = &tempPortRangeStart
					}
					if fwRaw["port_range_end"] != nil {
						tempPortRangeStart := fwRaw["port_range_end"].(int)
						firewall.Properties.PortRangeEnd = &tempPortRangeStart
					}
					if fwRaw["icmp_type"] != nil {
						tempIcmpType := fwRaw["icmp_type"].(string)
						if tempIcmpType != "" {
							i, _ := strconv.Atoi(tempIcmpType)
							firewall.Properties.IcmpType = &i
						}
					}
					if fwRaw["icmp_code"] != nil {
						tempIcmpCode := fwRaw["icmp_type"].(string)
						if tempIcmpCode != "" {
							i, _ := strconv.Atoi(tempIcmpCode)
							firewall.Properties.IcmpCode = &i
						}
					}

					request.Entities.Nics.Items[0].Entities = &profitbricks.NicEntities{
						FirewallRules: &profitbricks.FirewallRules{
							Items: []profitbricks.FirewallRule{
								firewall,
							},
						},
					}
				}
			}
		}
	}

	if len(request.Entities.Nics.Items[0].Properties.Ips) == 0 {
		request.Entities.Nics.Items[0].Properties.Ips = nil
	}
	server, err := connection.CreateServer(d.Get("datacenter_id").(string), request)

	jsn, _ := json.Marshal(request)
	log.Println("[DEBUG] Server request", string(jsn))
	log.Println("[DEBUG] Server response", server.Response)

	if err != nil {
		return fmt.Errorf(
			"Error creating server: (%s)", err)
	}

	// Wait, catching any errors
	_, errState := getStateChangeConf(meta, d, server.Headers.Get("Location"), schema.TimeoutCreate).WaitForState()
	if errState != nil {
		return errState
	}

	d.SetId(server.ID)
	server, err = connection.GetServer(d.Get("datacenter_id").(string), server.ID)
	if err != nil {
		return fmt.Errorf("Error fetching server: (%s)", err)
	}

	d.Set("primary_nic", server.Entities.Nics.Items[0].ID)
	if len(server.Entities.Nics.Items[0].Properties.Ips) > 0 {
		d.SetConnInfo(map[string]string{
			"type":     "ssh",
			"host":     server.Entities.Nics.Items[0].Properties.Ips[0],
			"password": request.Entities.Volumes.Items[0].Properties.ImagePassword,
		})
	}
	return resourceProfitBricksServerRead(d, meta)
}

func resourceProfitBricksServerRead(d *schema.ResourceData, meta interface{}) error {
	connection := meta.(*profitbricks.Client)
	dcId := d.Get("datacenter_id").(string)
	serverId := d.Id()

	server, err := connection.GetServer(dcId, serverId)
	if err != nil {
		if err2, ok := err.(profitbricks.ApiError); ok {
			if err2.HttpStatusCode() == 404 {
				d.SetId("")
				return nil
			}
		}
		return fmt.Errorf("Error occured while fetching a server ID %s %s", d.Id(), err)
	}
	d.Set("name", server.Properties.Name)
	d.Set("cores", server.Properties.Cores)
	d.Set("ram", server.Properties.RAM)
	d.Set("availability_zone", server.Properties.AvailabilityZone)

	if primarynic, ok := d.GetOk("primary_nic"); ok {
		d.Set("primary_nic", primarynic.(string))

		nic, err := connection.GetNic(dcId, serverId, primarynic.(string))
		if err != nil {
			return fmt.Errorf("Error occured while fetching nic %s for server ID %s %s", primarynic.(string), d.Id(), err)
		}

		if len(nic.Properties.Ips) > 0 {
			d.Set("primary_ip", nic.Properties.Ips[0])
		}

		if nRaw, ok := d.GetOk("nic"); ok {
			log.Printf("[DEBUG] parsing nic")

			nicRaw := nRaw.(*schema.Set).List()

			for _, raw := range nicRaw {

				rawMap := raw.(map[string]interface{})

				rawMap["lan"] = nic.Properties.Lan
				rawMap["name"] = nic.Properties.Name
				rawMap["dhcp"] = nic.Properties.Dhcp
				rawMap["nat"] = nic.Properties.Nat
				rawMap["firewall_active"] = nic.Properties.FirewallActive
				rawMap["ips"] = nic.Properties.Ips
			}
			d.Set("nic", nicRaw)
		}
	}

	if server.Properties.BootVolume != nil {
		d.Set("boot_volume", server.Properties.BootVolume.ID)
	}
	if server.Properties.BootCdrom != nil {
		d.Set("boot_cdrom", server.Properties.BootCdrom.ID)
	}
	return nil
}

func resourceProfitBricksServerUpdate(d *schema.ResourceData, meta interface{}) error {
	connection := meta.(*profitbricks.Client)
	dcId := d.Get("datacenter_id").(string)

	request := profitbricks.ServerProperties{}

	if d.HasChange("name") {
		_, n := d.GetChange("name")
		request.Name = n.(string)
	}
	if d.HasChange("cores") {
		_, n := d.GetChange("cores")
		request.Cores = n.(int)
	}
	if d.HasChange("ram") {
		_, n := d.GetChange("ram")
		request.RAM = n.(int)
	}
	if d.HasChange("availability_zone") {
		_, n := d.GetChange("availability_zone")
		request.AvailabilityZone = n.(string)
	}
	if d.HasChange("cpu_family") {
		_, n := d.GetChange("cpu_family")
		request.CPUFamily = n.(string)
	}
	server, err := connection.UpdateServer(dcId, d.Id(), request)

	if err != nil {
		return fmt.Errorf("Error occured while updating server ID %s %s", d.Id(), err)
	}
	//Volume stuff
	if d.HasChange("volume") {
		_, new := d.GetChange("volume")

		newVolume := new.(*schema.Set).List()
		properties := profitbricks.VolumeProperties{}

		for _, raw := range newVolume {
			rawMap := raw.(map[string]interface{})
			if rawMap["name"] != nil {
				properties.Name = rawMap["name"].(string)
			}
			if rawMap["size"] != nil {
				properties.Size = rawMap["size"].(int)
			}
			if rawMap["bus"] != nil {
				properties.Bus = rawMap["bus"].(string)
			}
		}

		resp, err := connection.UpdateVolume(d.Get("datacenter_id").(string), server.Entities.Volumes.Items[0].ID, properties)

		if err != nil {
			return fmt.Errorf("Error patching volume (%s) (%s)", d.Id(), err)
		}

		// Wait, catching any errors
		_, errState := getStateChangeConf(meta, d, resp.Headers.Get("Location"), schema.TimeoutUpdate).WaitForState()
		if errState != nil {
			return errState
		}
	}

	//Nic stuff
	if d.HasChange("nic") {
		nic := profitbricks.Nic{}
		for _, n := range server.Entities.Nics.Items {
			if n.ID == d.Get("primary_nic").(string) {
				nic = n
				break
			}
		}
		_, new := d.GetChange("nic")

		newNic := new.(*schema.Set).List()
		properties := profitbricks.NicProperties{}

		for _, raw := range newNic {
			rawMap := raw.(map[string]interface{})
			if rawMap["name"] != nil {
				properties.Name = rawMap["name"].(string)
			}
			if rawMap["ip"] != nil {
				rawIps := rawMap["ip"].(string)
				ips := strings.Split(rawIps, ",")

				if rawIps != "" {
					nic.Properties.Ips = ips
				}
			}
			if rawMap["lan"] != nil {
				properties.Lan = rawMap["lan"].(int)
			}
			if rawMap["dhcp"] != nil {
				val := rawMap["dhcp"].(bool)
				properties.Dhcp = &val
			}
			if rawMap["nat"] != nil {
				properties.Nat = rawMap["nat"].(bool)
			}
		}

		resp, err := connection.UpdateNic(d.Get("datacenter_id").(string), server.ID, nic.ID, properties)

		if err != nil {
			return fmt.Errorf(
				"Error updating nic (%s)", err)
		}

		// Wait, catching any errors
		_, errState := getStateChangeConf(meta, d, resp.Headers.Get("Location"), schema.TimeoutUpdate).WaitForState()
		if errState != nil {
			return errState
		}
	}

	return resourceProfitBricksServerRead(d, meta)
}

func resourceProfitBricksServerDelete(d *schema.ResourceData, meta interface{}) error {
	connection := meta.(*profitbricks.Client)
	dcId := d.Get("datacenter_id").(string)

	server, err := connection.GetServer(dcId, d.Id())

	if err != nil {
		return fmt.Errorf("Error occured while fetching a server ID %s %s", d.Id(), err)
	}

	if server.Properties.BootVolume != nil {
		resp, err := connection.DeleteVolume(dcId, server.Properties.BootVolume.ID)
		if err != nil {
			return fmt.Errorf("Error occured while delete volume %s of server ID %s %s", server.Properties.BootVolume.ID, d.Id(), err)
		}
		// Wait, catching any errors
		_, errState := getStateChangeConf(meta, d, resp.Get("Location"), schema.TimeoutDelete).WaitForState()
		if errState != nil {
			return errState
		}
	}

	resp, err := connection.DeleteServer(dcId, d.Id())
	if err != nil {
		return fmt.Errorf("An error occured while deleting a server ID %s %s", d.Id(), err)

	}

	// Wait, catching any errors
	_, errState := getStateChangeConf(meta, d, resp.Get("Location"), schema.TimeoutDelete).WaitForState()
	if errState != nil {
		return errState
	}

	d.SetId("")
	return nil
}

//Reads public key from file and returns key string iff valid
func readPublicKey(path string) (key string, err error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey(bytes)
	if err != nil {
		return "", err
	}
	return string(ssh.MarshalAuthorizedKey(pubKey)[:]), nil
}
