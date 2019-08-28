package main

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform/helper/schema"
	hyperkit "github.com/moby/hyperkit/go"
)

const (
	stateRunning = "running"
	stateStopped = "stopped"
)

var states = map[string]struct{}{
	stateRunning: {},
	stateStopped: {},
}

func resourceHyperKitVM() *schema.Resource {
	return &schema.Resource{
		Create: resourceHyperKitVMCreate,
		Read:   resourceHyperKitVMRead,
		Update: resourceHyperKitVMUpdate,
		Delete: resourceHyperKitVMDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"cpus": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1,
				ForceNew: true,
			},
			"memory": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1024,
				ForceNew: true,
			},
			"pid": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			"uuid": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"kernel": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"initrd": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"disk_image": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"path": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"size": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},
			"iso_images": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"state": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					if _, ok := states[v.(string)]; !ok {
						return nil, []error{fmt.Errorf("unknown state %s, must be one of %q", v, states)}
					}
					return
				},
			},
			"command_line": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func createHyperKitConfig(d *schema.ResourceData, m interface{}) (*hyperkit.HyperKit, error) {
	config := m.(Config)

	vm, err := hyperkit.New(config.HyperKitBin, config.VPNKitSocket, config.StateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance of hyperkit, %s", err)
	}

	vm.CPUs = d.Get("cpus").(int)
	vm.Memory = d.Get("memory").(int)
	vm.Kernel = d.Get("kernel").(string)
	vm.Initrd = d.Get("initrd").(string)
	vm.UUID = d.Id()
	vm.VPNKitUUID = d.Id()
	vm.Pid = d.Get("pid").(int)

	rawSet := d.Get("disk_image").(*schema.Set)
	for _, rawData := range rawSet.List() {
		data := rawData.(map[string]interface{})
		disk, err := hyperkit.NewDisk(data["path"].(string), data["size"].(int))
		if err != nil {
			return nil, fmt.Errorf("failed to create instance of hyperkit disk image, %s", err)
		}

		vm.Disks = append(vm.Disks, disk)
	}

	isoImages := []string{}
	for _, rawImage := range d.Get("iso_images").([]interface{}) {
		isoImages = append(isoImages, rawImage.(string))
	}
	vm.ISOImages = isoImages

	return vm, nil
}

func resourceHyperKitVMCreate(d *schema.ResourceData, m interface{}) error {
	if rawID, ok := d.GetOkExists("uuid"); ok {
		d.SetId(rawID.(string))
	} else {
		rawUUID, err := uuid.NewRandom()
		if err != nil {
			d.SetId("")
			return fmt.Errorf("failed to generate uuid, %s", err)
		}

		d.Set("uuid", rawUUID.String())
		d.SetId(rawUUID.String())
	}

	vm, err := createHyperKitConfig(d, m)
	if err != nil {
		d.SetId("")
		return fmt.Errorf("failed to create hyperkit instance, %s", err)
	}

	_, err = vm.Start(d.Get("command_line").(string))
	if err != nil {
		d.SetId("")
		return fmt.Errorf("failed to start hyperkit vm, %s", err)
	}
	d.Set("pid", vm.Pid)

	if d.Get("state").(string) == stateStopped {
		if err := vm.Stop(); err != nil {
			return fmt.Errorf("failed to stop hyperkit vm, %s", err)
		}
	}

	return resourceHyperKitVMRead(d, m)
}

func resourceHyperKitVMRead(d *schema.ResourceData, m interface{}) error {
	vm, err := createHyperKitConfig(d, m)
	if err != nil {
		return fmt.Errorf("failed to create hyperkit instance, %s", err)
	}

	if !vm.IsRunning() {
		d.Set("state", stateStopped)
	}

	return nil
}

func resourceHyperKitVMUpdate(d *schema.ResourceData, m interface{}) error {
	vm, err := createHyperKitConfig(d, m)
	if err != nil {
		return fmt.Errorf("failed to create hyperkit instance, %s", err)
	}

	if d.HasChange("state") {
		switch d.Get("state").(string) {
		case stateRunning:
			_, err := vm.Start(d.Get("command_line").(string))
			if err != nil {
				return fmt.Errorf("failed to start hyperkit vm, %s", err)
			}
			d.Set("state", stateRunning)
		case stateStopped:
			err = vm.Stop()
			if err != nil {
				return fmt.Errorf("failed to start hyperkit vm, %s", err)
			}
			d.Set("state", stateStopped)
		}
	}

	return resourceHyperKitVMRead(d, m)
}

func resourceHyperKitVMDelete(d *schema.ResourceData, m interface{}) error {
	vm, err := createHyperKitConfig(d, m)
	if err != nil {
		return fmt.Errorf("failed to create hyperkit instance, %s", err)
	}

	err = vm.Stop()
	if err != nil {
		return fmt.Errorf("failed to stop hyperkit vm, %s", err)
	}

	err = vm.Remove(true)
	if err != nil {
		return fmt.Errorf("failed to remove hyperkit state, %s", err)
	}

	return nil
}
