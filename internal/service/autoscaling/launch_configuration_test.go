package autoscaling_test

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/create"
	tfec2 "github.com/hashicorp/terraform-provider-aws/internal/service/ec2"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
)

func init() {
	resource.AddTestSweepers("aws_launch_configuration", &resource.Sweeper{
		Name:         "aws_launch_configuration",
		Dependencies: []string{"aws_autoscaling_group"},
		F:            testSweepLaunchConfigurations,
	})
}

func testSweepLaunchConfigurations(region string) error {
	client, err := acctest.SharedRegionalSweeperClient(region)
	if err != nil {
		return fmt.Errorf("error getting client: %s", err)
	}
	conn := client.(*conns.AWSClient).AutoScalingConn

	resp, err := conn.DescribeLaunchConfigurations(&autoscaling.DescribeLaunchConfigurationsInput{})
	if err != nil {
		if acctest.SkipSweepError(err) {
			log.Printf("[WARN] Skipping AutoScaling Launch Configuration sweep for %s: %s", region, err)
			return nil
		}
		return fmt.Errorf("Error retrieving launch configuration: %s", err)
	}

	if len(resp.LaunchConfigurations) == 0 {
		log.Print("[DEBUG] No aws launch configurations to sweep")
		return nil
	}

	for _, lc := range resp.LaunchConfigurations {
		name := *lc.LaunchConfigurationName

		log.Printf("[INFO] Deleting Launch Configuration: %s", name)
		_, err := conn.DeleteLaunchConfiguration(
			&autoscaling.DeleteLaunchConfigurationInput{
				LaunchConfigurationName: aws.String(name),
			})
		if err != nil {
			if tfawserr.ErrMessageContains(err, "InvalidConfiguration.NotFound", "") || tfawserr.ErrMessageContains(err, "ValidationError", "") {
				return nil
			}
			return err
		}
	}

	return nil
}

func TestAccAWSLaunchConfiguration_basic(t *testing.T) {
	var conf autoscaling.LaunchConfiguration
	resourceName := "aws_launch_configuration.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, autoscaling.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLaunchConfigurationConfig(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists(resourceName, &conf),
					acctest.MatchResourceAttrRegionalARN(resourceName, "arn", "autoscaling", regexp.MustCompile(`launchConfiguration:.+`)),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"associate_public_ip_address"},
			},
		},
	})
}

func TestAccAWSLaunchConfiguration_Name_Generated(t *testing.T) {
	var conf autoscaling.LaunchConfiguration
	resourceName := "aws_launch_configuration.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, autoscaling.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLaunchConfigurationConfigNameGenerated(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists(resourceName, &conf),
					create.TestCheckResourceAttrNameGenerated(resourceName, "name"),
					resource.TestCheckResourceAttr(resourceName, "name_prefix", "terraform-"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"associate_public_ip_address"},
			},
		},
	})
}

func TestAccAWSLaunchConfiguration_NamePrefix(t *testing.T) {
	var conf autoscaling.LaunchConfiguration
	resourceName := "aws_launch_configuration.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, autoscaling.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLaunchConfigurationConfigNamePrefix("tf-acc-test-prefix-"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists(resourceName, &conf),
					create.TestCheckResourceAttrNameFromPrefix(resourceName, "name", "tf-acc-test-prefix-"),
					resource.TestCheckResourceAttr(resourceName, "name_prefix", "tf-acc-test-prefix-"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"associate_public_ip_address"},
			},
		},
	})
}

func TestAccAWSLaunchConfiguration_withBlockDevices(t *testing.T) {
	var conf autoscaling.LaunchConfiguration
	resourceName := "aws_launch_configuration.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, autoscaling.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLaunchConfigurationConfig(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists(resourceName, &conf),
					testAccCheckAWSLaunchConfigurationAttributes(&conf),
					resource.TestMatchResourceAttr(resourceName, "image_id", regexp.MustCompile("^ami-[0-9a-z]+")),
					resource.TestCheckResourceAttr(resourceName, "instance_type", "m1.small"),
					resource.TestCheckResourceAttr(resourceName, "associate_public_ip_address", "true"),
					resource.TestCheckResourceAttr(resourceName, "spot_price", ""),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"associate_public_ip_address"},
			},
		},
	})
}

func TestAccAWSLaunchConfiguration_withInstanceStoreAMI(t *testing.T) {
	var conf autoscaling.LaunchConfiguration
	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_launch_configuration.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, autoscaling.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLaunchConfigurationConfigWithInstanceStoreAMI(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists(resourceName, &conf),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"associate_public_ip_address"},
			},
		},
	})
}

func TestAccAWSLaunchConfiguration_RootBlockDevice_AmiDisappears(t *testing.T) {
	var ami ec2.Image
	var conf autoscaling.LaunchConfiguration
	amiCopyResourceName := "aws_ami_copy.test"
	resourceName := "aws_launch_configuration.test"
	rName := sdkacctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, autoscaling.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLaunchConfigurationConfigWithRootBlockDeviceCopiedAmi(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists(resourceName, &conf),
					testAccCheckAmiExists(amiCopyResourceName, &ami),
					acctest.CheckResourceDisappears(acctest.Provider, tfec2.ResourceAMI(), amiCopyResourceName),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccAWSLaunchConfigurationConfigWithRootBlockDeviceVolumeSize(rName, 10),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists(resourceName, &conf),
				),
			},
		},
	})
}

func TestAccAWSLaunchConfiguration_RootBlockDevice_VolumeSize(t *testing.T) {
	var conf autoscaling.LaunchConfiguration
	resourceName := "aws_launch_configuration.test"
	rName := sdkacctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, autoscaling.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLaunchConfigurationConfigWithRootBlockDeviceVolumeSize(rName, 11),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "root_block_device.0.volume_size", "11"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"associate_public_ip_address"},
			},
			{
				Config: testAccAWSLaunchConfigurationConfigWithRootBlockDeviceVolumeSize(rName, 20),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "root_block_device.0.volume_size", "20"),
				),
			},
		},
	})
}

func TestAccAWSLaunchConfiguration_encryptedRootBlockDevice(t *testing.T) {
	var conf autoscaling.LaunchConfiguration
	rInt := sdkacctest.RandInt()
	resourceName := "aws_launch_configuration.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, autoscaling.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLaunchConfigurationConfigWithEncryptedRootBlockDevice(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "root_block_device.0.encrypted", "true"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"associate_public_ip_address"},
			},
		},
	})
}

func TestAccAWSLaunchConfiguration_withSpotPrice(t *testing.T) {
	var conf autoscaling.LaunchConfiguration
	resourceName := "aws_launch_configuration.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, autoscaling.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLaunchConfigurationWithSpotPriceConfig(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spot_price", "0.05"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"associate_public_ip_address"},
			},
		},
	})
}

func TestAccAWSLaunchConfiguration_withVpcClassicLink(t *testing.T) {
	var vpc ec2.Vpc
	var group ec2.SecurityGroup
	var conf autoscaling.LaunchConfiguration
	rInt := sdkacctest.RandInt()
	resourceName := "aws_launch_configuration.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); acctest.PreCheckEC2Classic(t) },
		ErrorCheck:   acctest.ErrorCheck(t, autoscaling.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLaunchConfigurationConfig_withVpcClassicLink(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists(resourceName, &conf),
					acctest.CheckVPCExists("aws_vpc.test", &vpc),
					testAccCheckAWSSecurityGroupExists("aws_security_group.test", &group),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSLaunchConfiguration_withIAMProfile(t *testing.T) {
	var conf autoscaling.LaunchConfiguration
	rInt := sdkacctest.RandInt()
	resourceName := "aws_launch_configuration.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, autoscaling.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLaunchConfigurationConfig_withIAMProfile(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists(resourceName, &conf),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"associate_public_ip_address"},
			},
		},
	})
}

func TestAccAWSLaunchConfiguration_withEncryption(t *testing.T) {
	var conf autoscaling.LaunchConfiguration
	resourceName := "aws_launch_configuration.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, autoscaling.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLaunchConfigurationWithEncryption(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists("aws_launch_configuration.test", &conf),
					testAccCheckAWSLaunchConfigurationWithEncryption(&conf),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"associate_public_ip_address"},
			},
		},
	})
}

func TestAccAWSLaunchConfiguration_withGP3(t *testing.T) {
	var conf autoscaling.LaunchConfiguration
	resourceName := "aws_launch_configuration.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, autoscaling.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLaunchConfigurationWithGP3(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists("aws_launch_configuration.test", &conf),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ebs_block_device.*", map[string]string{
						"volume_type": "gp3",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ebs_block_device.*", map[string]string{
						"throughput": "150",
					}),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"associate_public_ip_address"},
			},
		},
	})
}

func TestAccAWSLaunchConfiguration_updateEbsBlockDevices(t *testing.T) {
	var conf autoscaling.LaunchConfiguration
	resourceName := "aws_launch_configuration.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, autoscaling.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLaunchConfigurationWithEncryption(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists(resourceName, &conf),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ebs_block_device.*", map[string]string{
						"volume_size": "9",
					}),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"associate_public_ip_address"},
			},
			{
				Config: testAccAWSLaunchConfigurationWithEncryptionUpdated(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists(resourceName, &conf),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ebs_block_device.*", map[string]string{
						"volume_size": "10",
					}),
				),
			},
		},
	})
}

func TestAccAWSLaunchConfiguration_metadataOptions(t *testing.T) {
	var conf autoscaling.LaunchConfiguration
	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_launch_configuration.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, autoscaling.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLaunchConfigurationConfigMetadataOptions(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata_options.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "metadata_options.0.http_endpoint", "enabled"),
					resource.TestCheckResourceAttr(resourceName, "metadata_options.0.http_tokens", "required"),
					resource.TestCheckResourceAttr(resourceName, "metadata_options.0.http_put_response_hop_limit", "2"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSLaunchConfiguration_ebs_noDevice(t *testing.T) {
	var conf autoscaling.LaunchConfiguration
	rInt := sdkacctest.RandInt()
	resourceName := "aws_launch_configuration.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, autoscaling.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLaunchConfigurationConfigEbsNoDevice(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "ebs_block_device.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ebs_block_device.*", map[string]string{
						"device_name": "/dev/sda2",
						"no_device":   "true",
					}),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"associate_public_ip_address"},
			},
		},
	})
}

func TestAccAWSLaunchConfiguration_userData(t *testing.T) {
	var conf autoscaling.LaunchConfiguration
	resourceName := "aws_launch_configuration.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, autoscaling.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLaunchConfigurationConfig_userData(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "user_data", "3dc39dda39be1205215e776bad998da361a5955d"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"associate_public_ip_address"},
			},
			{
				Config: testAccAWSLaunchConfigurationConfig_userDataBase64(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "user_data_base64", "aGVsbG8gd29ybGQ="),
				),
			},
		},
	})
}

func testAccCheckAWSLaunchConfigurationWithEncryption(conf *autoscaling.LaunchConfiguration) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Map out the block devices by name, which should be unique.
		blockDevices := make(map[string]*autoscaling.BlockDeviceMapping)
		for _, blockDevice := range conf.BlockDeviceMappings {
			blockDevices[*blockDevice.DeviceName] = blockDevice
		}

		// Check if the root block device exists.
		if _, ok := blockDevices["/dev/xvda"]; !ok {
			return fmt.Errorf("block device doesn't exist: /dev/xvda")
		} else if blockDevices["/dev/xvda"].Ebs.Encrypted != nil {
			return fmt.Errorf("root device should not include value for Encrypted")
		}

		// Check if the secondary block device exists.
		if _, ok := blockDevices["/dev/sdb"]; !ok {
			return fmt.Errorf("block device doesn't exist: /dev/sdb")
		} else if !*blockDevices["/dev/sdb"].Ebs.Encrypted {
			return fmt.Errorf("block device isn't encrypted as expected: /dev/sdb")
		}

		return nil
	}
}

func testAccCheckAWSLaunchConfigurationDestroy(s *terraform.State) error {
	conn := acctest.Provider.Meta().(*conns.AWSClient).AutoScalingConn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_launch_configuration" {
			continue
		}

		describe, err := conn.DescribeLaunchConfigurations(
			&autoscaling.DescribeLaunchConfigurationsInput{
				LaunchConfigurationNames: []*string{aws.String(rs.Primary.ID)},
			})

		if err == nil {
			if len(describe.LaunchConfigurations) != 0 &&
				*describe.LaunchConfigurations[0].LaunchConfigurationName == rs.Primary.ID {
				return fmt.Errorf("Launch Configuration still exists")
			}
		}

		// Verify the error
		if !tfawserr.ErrMessageContains(err, "InvalidLaunchConfiguration.NotFound", "") {
			return err
		}
	}

	return nil
}

func testAccCheckAWSLaunchConfigurationAttributes(conf *autoscaling.LaunchConfiguration) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if !strings.HasPrefix(*conf.LaunchConfigurationName, "terraform-") && !strings.HasPrefix(*conf.LaunchConfigurationName, "tf-acc-test-") {
			return fmt.Errorf("Bad name: %s", *conf.LaunchConfigurationName)
		}

		if *conf.InstanceType != "m1.small" {
			return fmt.Errorf("Bad instance_type: %s", *conf.InstanceType)
		}

		// Map out the block devices by name, which should be unique.
		blockDevices := make(map[string]*autoscaling.BlockDeviceMapping)
		for _, blockDevice := range conf.BlockDeviceMappings {
			blockDevices[*blockDevice.DeviceName] = blockDevice
		}

		// Check if the root block device exists.
		if _, ok := blockDevices["/dev/xvda"]; !ok {
			return fmt.Errorf("block device doesn't exist: /dev/xvda")
		}

		// Check if the secondary block device exists.
		if _, ok := blockDevices["/dev/sdb"]; !ok {
			return fmt.Errorf("block device doesn't exist: /dev/sdb")
		}

		// Check if the third block device exists.
		if _, ok := blockDevices["/dev/sdc"]; !ok {
			return fmt.Errorf("block device doesn't exist: /dev/sdc")
		}

		// Check if the fourth block device exists.
		if _, ok := blockDevices["/dev/sde"]; !ok {
			return fmt.Errorf("block device doesn't exist: /dev/sde")
		}

		return nil
	}
}

func testAccCheckAWSLaunchConfigurationExists(n string, res *autoscaling.LaunchConfiguration) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Launch Configuration ID is set")
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).AutoScalingConn

		describeOpts := autoscaling.DescribeLaunchConfigurationsInput{
			LaunchConfigurationNames: []*string{aws.String(rs.Primary.ID)},
		}
		describe, err := conn.DescribeLaunchConfigurations(&describeOpts)

		if err != nil {
			return err
		}

		if len(describe.LaunchConfigurations) != 1 ||
			*describe.LaunchConfigurations[0].LaunchConfigurationName != rs.Primary.ID {
			return fmt.Errorf("Launch Configuration Group not found")
		}

		*res = *describe.LaunchConfigurations[0]

		return nil
	}
}

func testAccAWSLaunchConfigurationConfigWithInstanceStoreAMI(rName string) string {
	return acctest.ConfigCompose(testAccLatestAmazonLinuxPvInstanceStoreAmiConfig(), fmt.Sprintf(`
resource "aws_launch_configuration" "test" {
  name     = %[1]q
  image_id = data.aws_ami.amzn-ami-minimal-pv-instance-store.id

  # When the instance type is updated, the new type must support ephemeral storage.
  instance_type = "m1.small"
}
`, rName))
}

func testAccAWSLaunchConfigurationConfigWithRootBlockDeviceCopiedAmi(rName string) string {
	return acctest.ConfigCompose(acctest.ConfigLatestAmazonLinuxHVMEBSAMI(), fmt.Sprintf(`
data "aws_region" "current" {}

resource "aws_ami_copy" "test" {
  name              = %[1]q
  source_ami_id     = data.aws_ami.amzn-ami-minimal-hvm-ebs.id
  source_ami_region = data.aws_region.current.name
}

resource "aws_launch_configuration" "test" {
  name          = %[1]q
  image_id      = aws_ami_copy.test.id
  instance_type = "t3.micro"

  root_block_device {
    volume_size = 10
  }
}
`, rName))
}

func testAccAWSLaunchConfigurationConfigWithRootBlockDeviceVolumeSize(rName string, volumeSize int) string {
	return acctest.ConfigCompose(acctest.ConfigLatestAmazonLinuxHVMEBSAMI(), fmt.Sprintf(`
resource "aws_launch_configuration" "test" {
  name          = %[1]q
  image_id      = data.aws_ami.amzn-ami-minimal-hvm-ebs.id
  instance_type = "t3.micro"

  root_block_device {
    volume_size = %[2]d
  }
}
`, rName, volumeSize))
}

func testAccAWSLaunchConfigurationConfigWithEncryptedRootBlockDevice(rInt int) string {
	return acctest.ConfigCompose(
		acctest.ConfigAvailableAZsNoOptIn(),
		acctest.ConfigLatestAmazonLinuxHVMEBSAMI(),
		fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = "terraform-testacc-instance-%[1]d"
  }
}

resource "aws_subnet" "test" {
  cidr_block        = "10.1.1.0/24"
  vpc_id            = aws_vpc.test.id
  availability_zone = data.aws_availability_zones.available.names[0]

  tags = {
    Name = "terraform-testacc-instance-%[1]d"
  }
}

resource "aws_launch_configuration" "test" {
  name_prefix                 = "tf-acc-test-%[1]d"
  image_id                    = data.aws_ami.amzn-ami-minimal-hvm-ebs.id
  instance_type               = "t3.nano"
  user_data                   = "testtest-user-data"
  associate_public_ip_address = true

  root_block_device {
    encrypted   = true
    volume_type = "gp2"
    volume_size = 11
  }
}
`, rInt))
}

func testAccAWSLaunchConfigurationConfigMetadataOptions(rName string) string {
	return acctest.ConfigCompose(
		acctest.ConfigLatestAmazonLinuxHVMEBSAMI(),
		fmt.Sprintf(`
resource "aws_launch_configuration" "test" {
  image_id      = data.aws_ami.amzn-ami-minimal-hvm-ebs.id
  instance_type = "t3.nano"
  name          = %[1]q
  metadata_options {
    http_endpoint               = "enabled"
    http_tokens                 = "required"
    http_put_response_hop_limit = 2
  }
}
`, rName))
}

func testAccAWSLaunchConfigurationConfig() string {
	return acctest.ConfigCompose(acctest.ConfigLatestAmazonLinuxHVMEBSAMI(), fmt.Sprintf(`
resource "aws_launch_configuration" "test" {
  name                        = "tf-acc-test-%d"
  image_id                    = data.aws_ami.amzn-ami-minimal-hvm-ebs.id
  instance_type               = "m1.small"
  user_data                   = "testtest-user-data"
  associate_public_ip_address = true

  root_block_device {
    volume_type = "gp2"
    volume_size = 11
  }

  ebs_block_device {
    device_name = "/dev/sdb"
    volume_size = 9
  }

  ebs_block_device {
    device_name = "/dev/sdc"
    volume_size = 10
    volume_type = "io1"
    iops        = 100
  }

  ephemeral_block_device {
    device_name  = "/dev/sde"
    virtual_name = "ephemeral0"
  }
}
`, sdkacctest.RandInt()))
}

func testAccAWSLaunchConfigurationWithSpotPriceConfig() string {
	return acctest.ConfigCompose(acctest.ConfigLatestAmazonLinuxHVMEBSAMI(), fmt.Sprintf(`
resource "aws_launch_configuration" "test" {
  name          = "tf-acc-test-%d"
  image_id      = data.aws_ami.amzn-ami-minimal-hvm-ebs.id
  instance_type = "t2.micro"
  spot_price    = "0.05"
}
`, sdkacctest.RandInt()))
}

func testAccAWSLaunchConfigurationConfigNameGenerated() string {
	return acctest.ConfigCompose(acctest.ConfigLatestAmazonLinuxHVMEBSAMI(), `
resource "aws_launch_configuration" "test" {
  image_id      = data.aws_ami.amzn-ami-minimal-hvm-ebs.id
  instance_type = "t2.micro"
}
`)
}

func testAccAWSLaunchConfigurationConfigNamePrefix(namePrefix string) string {
	return acctest.ConfigCompose(acctest.ConfigLatestAmazonLinuxHVMEBSAMI(), fmt.Sprintf(`
resource "aws_launch_configuration" "test" {
  name_prefix   = %[1]q
  image_id      = data.aws_ami.amzn-ami-minimal-hvm-ebs.id
  instance_type = "t2.micro"
}
`, namePrefix))
}

func testAccAWSLaunchConfigurationWithEncryption() string {
	return acctest.ConfigCompose(acctest.ConfigLatestAmazonLinuxHVMEBSAMI(), `
resource "aws_launch_configuration" "test" {
  image_id                    = data.aws_ami.amzn-ami-minimal-hvm-ebs.id
  instance_type               = "t2.micro"
  associate_public_ip_address = false

  root_block_device {
    volume_type = "gp2"
    volume_size = 11
  }

  ebs_block_device {
    device_name = "/dev/sdb"
    volume_size = 9
    encrypted   = true
  }
}
`)
}

func testAccAWSLaunchConfigurationWithGP3() string {
	return acctest.ConfigCompose(acctest.ConfigLatestAmazonLinuxHVMEBSAMI(), `
resource "aws_launch_configuration" "test" {
  image_id                    = data.aws_ami.amzn-ami-minimal-hvm-ebs.id
  instance_type               = "t2.micro"
  associate_public_ip_address = false

  root_block_device {
    volume_type = "gp3"
    volume_size = 11
  }

  ebs_block_device {
    volume_type = "gp3"
    device_name = "/dev/sdb"
    volume_size = 9
    encrypted   = true
    throughput  = 150
  }
}
`)
}

func testAccAWSLaunchConfigurationWithEncryptionUpdated() string {
	return acctest.ConfigCompose(acctest.ConfigLatestAmazonLinuxHVMEBSAMI(), `
resource "aws_launch_configuration" "test" {
  image_id                    = data.aws_ami.amzn-ami-minimal-hvm-ebs.id
  instance_type               = "t2.micro"
  associate_public_ip_address = false

  root_block_device {
    volume_type = "gp2"
    volume_size = 11
  }

  ebs_block_device {
    device_name = "/dev/sdb"
    volume_size = 10
    encrypted   = true
  }
}
`)
}

func testAccAWSLaunchConfigurationConfig_withVpcClassicLink(rInt int) string {
	return acctest.ConfigCompose(acctest.ConfigLatestAmazonLinuxHVMEBSAMI(), fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block         = "10.0.0.0/16"
  enable_classiclink = true

  tags = {
    Name = "terraform-testacc-launch-configuration-with-vpc-classic-link"
  }
}

resource "aws_security_group" "test" {
  name   = "tf-acc-test-%[1]d"
  vpc_id = aws_vpc.test.id
}

resource "aws_launch_configuration" "test" {
  name          = "tf-acc-test-%[1]d"
  image_id      = data.aws_ami.amzn-ami-minimal-hvm-ebs.id
  instance_type = "t2.micro"

  vpc_classic_link_id              = aws_vpc.test.id
  vpc_classic_link_security_groups = [aws_security_group.test.id]
}
`, rInt))
}

func testAccAWSLaunchConfigurationConfig_withIAMProfile(rInt int) string {
	return acctest.ConfigCompose(acctest.ConfigLatestAmazonLinuxHVMEBSAMI(), fmt.Sprintf(`
resource "aws_iam_role" "role" {
  name = "tf-acc-test-%[1]d"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_iam_instance_profile" "profile" {
  name = "tf-acc-test-%[1]d"
  role = aws_iam_role.role.name
}

resource "aws_launch_configuration" "test" {
  image_id             = data.aws_ami.amzn-ami-minimal-hvm-ebs.id
  instance_type        = "t2.nano"
  iam_instance_profile = aws_iam_instance_profile.profile.name
}
`, rInt))
}

func testAccAWSLaunchConfigurationConfigEbsNoDevice(rInt int) string {
	return acctest.ConfigCompose(acctest.ConfigLatestAmazonLinuxHVMEBSAMI(), fmt.Sprintf(`
resource "aws_launch_configuration" "test" {
  name_prefix   = "tf-acc-test-%d"
  image_id      = data.aws_ami.amzn-ami-minimal-hvm-ebs.id
  instance_type = "m1.small"

  ebs_block_device {
    device_name = "/dev/sda2"
    no_device   = true
  }
}
`, rInt))
}

func testAccAWSLaunchConfigurationConfig_userData() string {
	return acctest.ConfigCompose(acctest.ConfigLatestAmazonLinuxHVMEBSAMI(), `
resource "aws_launch_configuration" "test" {
  image_id                    = data.aws_ami.amzn-ami-minimal-hvm-ebs.id
  instance_type               = "t2.micro"
  user_data                   = "foo:-with-character's"
  associate_public_ip_address = false
}
`)
}

func testAccAWSLaunchConfigurationConfig_userDataBase64() string {
	return acctest.ConfigCompose(acctest.ConfigLatestAmazonLinuxHVMEBSAMI(), `
resource "aws_launch_configuration" "test" {
  image_id                    = data.aws_ami.amzn-ami-minimal-hvm-ebs.id
  instance_type               = "t2.micro"
  user_data_base64            = base64encode("hello world")
  associate_public_ip_address = false
}
`)
}

func testAccCheckAmiExists(n string, ami *ec2.Image) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("AMI Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No AMI ID is set")
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn

		var resp *ec2.DescribeImagesOutput
		err := resource.Retry(1*time.Minute, func() *resource.RetryError {
			opts := &ec2.DescribeImagesInput{
				ImageIds: []*string{aws.String(rs.Primary.ID)},
			}
			var err error
			resp, err = conn.DescribeImages(opts)
			if err != nil {
				// This can be just eventual consistency
				if tfawserr.ErrMessageContains(err, "InvalidAMIID.NotFound", "") {
					return resource.RetryableError(err)
				}

				return resource.NonRetryableError(err)
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("Unable to find AMI after retries: %s", err)
		}

		if len(resp.Images) == 0 {
			return fmt.Errorf("AMI not found")
		}
		*ami = *resp.Images[0]
		return nil
	}
}

// testAccLatestAmazonLinuxPvInstanceStoreAmiConfig returns the configuration for a data source that
// describes the latest Amazon Linux AMI using PV virtualization and an instance store root device.
// The data source is named 'amzn-ami-minimal-pv-ebs'.
func testAccLatestAmazonLinuxPvInstanceStoreAmiConfig() string {
	return `
data "aws_ami" "amzn-ami-minimal-pv-instance-store" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["amzn-ami-minimal-pv-*"]
  }

  filter {
    name   = "root-device-type"
    values = ["instance-store"]
  }
}
`
}

func testAccCheckAWSSecurityGroupExists(n string, group *ec2.SecurityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Security Group is set")
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn

		sg, err := tfec2.FindSecurityGroupByID(conn, rs.Primary.ID)
		if tfresource.NotFound(err) {
			return fmt.Errorf("Security Group (%s) not found: %w", rs.Primary.ID, err)
		}
		if err != nil {
			return err
		}

		*group = *sg

		return nil
	}
}
