package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/alecthomas/kong"
)

var Version = "dev"
var Revision = "HEAD"

type GlobalOptions struct {
}

type CLI struct {
	Backup struct {
		Pattern     string `help:"Regular expression pattern to filter user pools" default:".*"`
		URI         string `help:"Backup destination URI (e.g., s3://bucket/prefix/file.tar.gz or file:///path/to/backup.tar.gz)" required:""`
		KMSRegion   string `help:"KMS region (e.g., ap-northeast-1)" default:"ap-northeast-1"`
		KMSKeyID    string `help:"KMS key ID (e.g., alias/my-key or arn:aws:kms:region:account:key/key-id)" and:"KMSKeyID,DataKeyPath"`
		DataKeyPath string `help:"Data key file path (e.g., file:///path/to/datakey.json)" and:"KMSKeyID,DataKeyPath"`
	} `cmd:"" help:"Backup Cognito user pools"`

	List struct {
		Pattern string `help:"Regular expression pattern to filter user pools" default:".*"`
	} `cmd:"" help:"List Cognito user pools"`

	Restore struct {
		Pattern   string `help:"Regular expression pattern to filter backup files" default:".*"`
		URI       string `help:"Backup source URI (e.g., s3://bucket/prefix/file.tar.gz or file:///path/to/backup.tar.gz)" required:""`
		KMSRegion string `help:"KMS region (e.g., ap-northeast-1)" default:"ap-northeast-1"`
	} `cmd:"" help:"Restore Cognito user pools from backup"`

	Decrypt struct {
		Input       string `help:"Path to encrypted backup file" required:""`
		Output      string `help:"Path to output decrypted backup file" required:""`
		KMSRegion   string `help:"KMS region (e.g., ap-northeast-1)" default:"ap-northeast-1"`
		KMSKeyID    string `help:"KMS key ID (e.g., alias/my-key or arn:aws:kms:region:account:key/key-id)" required:""`
		DataKeyPath string `help:"Data key file path (e.g., file:///path/to/datakey.json)" required:""`
	} `cmd:"" help:"Decrypt encrypted backup file"`

	GenerateDatakey struct {
		KMSRegion string `help:"KMS region (e.g., ap-northeast-1)" default:"ap-northeast-1"`
		KMSKeyID  string `help:"KMS key ID for generating data key" required:""`
		Output    string `help:"Output file path (default: stdout)"`
		Format    string `help:"Output format (json|base64)" default:"json" enum:"json,base64"`
		Spec      string `help:"Data key specification (AES_256|AES_128)" default:"AES_256" enum:"AES_256,AES_128"`
		Test      bool   `help:"Generate test data key without KMS call"`
	} `cmd:"generate-datakey" help:"Generate KMS data key for encryption"`
	Version VersionFlag `name:"version" help:"show version"`
}

type VersionFlag string

func (v VersionFlag) Decode(ctx *kong.DecodeContext) error { return nil }
func (v VersionFlag) IsBool() bool                         { return true }
func (v VersionFlag) BeforeApply(app *kong.Kong, vars kong.Vars) error {
	fmt.Printf("%s-%s\n", Version, Revision)
	app.Exit(0)
	return nil
}

func RunCLI(ctx context.Context, args []string) error {
	cli := CLI{
		Version: VersionFlag("0.1.0"),
	}
	parser, err := kong.New(&cli)
	if err != nil {
		return fmt.Errorf("error creating CLI parser: %w", err)
	}
	kctx, err := parser.Parse(args)
	if err != nil {
		fmt.Printf("error parsing CLI: %v\n", err)
		return fmt.Errorf("error parsing CLI: %w", err)
	}
	cmd := strings.Fields(kctx.Command())[0]
	if cmd == "version" {
		fmt.Println(Version)
		return nil
	}

	switch cmd {
	case "backup":
		return Backup(&cli)
	case "list":
		return List(&cli)
	case "restore":
		return Restore(&cli)
	case "decrypt":
		return Decrypt(&cli)
	case "generate-datakey":
		return GenerateDatakey(&cli)
	}
	return nil
}
