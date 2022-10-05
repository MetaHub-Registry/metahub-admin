package minio

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	layerBucket     = "layers"
	manifestsBucket = "manifests"
	configsBucket   = "configs"
)

type BucketConfig struct {
	Client   *minio.Client
	URL      string
	Region   string
	Bucket   string
	User     string
	Password string
	Redirect bool
	Ssl      bool
}

type Minio struct {
	BucketConfig map[string]BucketConfig
}

func NewMinio(cfgFile string) (m Minio, err error) {
	m = Minio{
		BucketConfig: map[string]BucketConfig{},
	}
	viper.SetConfigType("yaml")
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".cobra-example" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".metahub-admin")
		viper.SetConfigType("yaml")
	}
	m.BucketConfig = map[string]BucketConfig{
		layerBucket:     {},
		manifestsBucket: {},
		configsBucket:   {},
	}
	bc := BucketConfig{
		URL:      viper.GetString("minio.url"),
		Region:   viper.GetString("minio.region"),
		User:     viper.GetString("minio.user"),
		Password: viper.GetString("minio.password"),
		Redirect: viper.GetBool("minio.redirect"),
		Ssl:      viper.GetBool("minio.ssl"),
	}
	for _, v := range reflect.ValueOf(m.BucketConfig).MapKeys() {
		b := fmt.Sprintf("%s", v)
		bCfg := BucketConfig{
			Bucket: b,
		}
		key := fmt.Sprintf("minio.%s.url", b)
		if viper.IsSet(key) {
			bCfg.URL = viper.GetString(key)
		} else {
			bCfg.URL = bc.URL
		}
		key = fmt.Sprintf("minio.%s.region", b)
		if viper.IsSet(key) {
			bCfg.Region = viper.GetString(key)
		} else {
			bCfg.Region = bc.Region
		}
		key = fmt.Sprintf("minio.%s.user", b)
		if viper.IsSet(key) {
			bCfg.User = viper.GetString(key)
		} else {
			bCfg.User = bc.User
		}
		key = fmt.Sprintf("minio.%s.password", b)
		if viper.IsSet(key) {
			bCfg.Password = viper.GetString(key)
		} else {
			bCfg.Password = bc.Password
		}
		key = fmt.Sprintf("minio.%s.redirect", b)
		if viper.IsSet(key) {
			bCfg.Redirect = viper.GetBool(key)
		} else {
			bCfg.Redirect = bc.Redirect
		}
		key = fmt.Sprintf("minio.%s.ssl", b)
		if viper.IsSet(key) {
			bCfg.Ssl = viper.GetBool(key)
		} else {
			bCfg.Ssl = bc.Ssl
		}
		m.BucketConfig[b] = bCfg
	}
	err = m.ConnectClients()
	if err != nil {
		logrus.Error(err)
	}
	return
}

func (m Minio) ConnectClients() (err error) {
	for _, v := range reflect.ValueOf(m.BucketConfig).MapKeys() {
		b := fmt.Sprintf("%s", v)
		var bCfg BucketConfig
		if bc, ok := m.BucketConfig[b]; !ok {
			logrus.Errorf("Bucket '%s' not found in config", b)
		} else {
			bCfg = bc
		}
		logrus.Infof("Connecting to minio bucket '%s'", bCfg.Bucket)

		cli, e := minio.New(bCfg.URL, &minio.Options{
			Creds:  credentials.NewStaticV4(bCfg.User, bCfg.Password, ""),
			Secure: bCfg.Ssl,
			Region: bCfg.Region,
		})
		if e != nil {
			logrus.Errorf("Error creating minio client: %v\n %s\n", bCfg, e)
			err = e
			return
		}
		bCfg.Client = cli
		m.BucketConfig[b] = bCfg
	}
	return
}

func (m Minio) BucketCheck() (err error) {
	ctx := context.Background()
	for _, bc := range m.BucketConfig {
		logrus.Info("Checking bucket ", bc.Bucket)
		exists, err := bc.Client.BucketExists(ctx, bc.Bucket)
		if err != nil {
			logrus.Errorf("Error checking bucket %s: %s\n", bc.Bucket, err)
			return err
		}
		if !exists {
			err = bc.Client.MakeBucket(ctx, bc.Bucket, minio.MakeBucketOptions{Region: bc.Region})
			if err != nil {
				logrus.Errorf(err.Error())
				return err
			}
		}
	}
	return
}

func (m Minio) StatObj(ctx context.Context, bucket, object string) (obj minio.ObjectInfo, err error) {
	//logrus.Warnf("ConfigStatObj in bucket '%s': %s", bucket, object)
	cli := m.BucketConfig[bucket].Client
	return cli.StatObject(ctx, bucket, object, minio.StatObjectOptions{})
}

func (m Minio) ManifestsStatObj(ctx context.Context, object string) (obj minio.ObjectInfo, err error) {
	return m.StatObj(ctx, manifestsBucket, object)
}

func (m Minio) ConfigStatObj(ctx context.Context, object string) (obj minio.ObjectInfo, err error) {
	return m.StatObj(ctx, configsBucket, object)
}

func (m Minio) ManifestsPutObj(ctx context.Context, object string, data []byte) (err error) {
	cli := m.BucketConfig[manifestsBucket].Client
	_, err = cli.PutObject(ctx, manifestsBucket, object, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{})
	return
}
