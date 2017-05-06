package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	data "github.com/tendermint/go-wire/data"
	"github.com/tendermint/go-wire/data/base58"
	cmn "github.com/tendermint/tmlibs/common"
)

const (
	RootFlag     = "root"
	OutputFlag   = "output"
	EncodingFlag = "encoding"
)

func PrepareMainCmd(cmd *cobra.Command, envPrefix, defautRoot string) func() {
	cobra.OnInitialize(func() { initEnv(envPrefix) })
	cmd.PersistentFlags().StringP(RootFlag, "r", defautRoot, "root directory for config and data")
	cmd.PersistentFlags().StringP(EncodingFlag, "e", "hex", "Binary encoding (hex|b64|btc)")
	cmd.PersistentFlags().StringP(OutputFlag, "o", "text", "Output format (text|json)")
	cmd.PersistentPreRunE = multiE(bindFlags, setEncoding, validateOutput, cmd.PersistentPreRunE)
	return func() { Execute(cmd) }
}

// initEnv sets to use ENV variables if set.
func initEnv(prefix string) {
	// env variables with TM prefix (eg. TM_ROOT)
	viper.SetEnvPrefix(prefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
}

// Execute calls cmd.Execute and exits if there is an error.
func Execute(cmd *cobra.Command) {
	if err := cmd.Execute(); err != nil {
		cmn.Exit(err.Error())
	}
}

//Add debugging flag and execute the root command
func ExecuteWithDebug(RootCmd *cobra.Command) {

	var debug bool
	RootCmd.SilenceUsage = true
	RootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enables stack trace error messages")

	//note that Execute() prints the error if encountered, so no need to reprint the error,
	//  only if we want the full stack trace
	if err := RootCmd.Execute(); err != nil && debug {
		cmn.Exit(fmt.Sprintf("%+v\n", err))
	}
}

type wrapE func(cmd *cobra.Command, args []string) error

func multiE(fs ...wrapE) wrapE {
	return func(cmd *cobra.Command, args []string) error {
		for _, f := range fs {
			if f != nil {
				if err := f(cmd, args); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func bindFlags(cmd *cobra.Command, args []string) error {
	// cmd.Flags() includes flags from this command and all persistent flags from the parent
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		return err
	}

	// rootDir is command line flag, env variable, or default $HOME/.tlc
	rootDir := viper.GetString("root")
	viper.SetConfigName("config") // name of config file (without extension)
	viper.AddConfigPath(rootDir)  // search root directory

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		// stderr, so if we redirect output to json file, this doesn't appear
		// fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	} else if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
		// we ignore not found error, only parse error
		// stderr, so if we redirect output to json file, this doesn't appear
		fmt.Fprintf(os.Stderr, "%#v", err)
	}
	return nil
}

// setEncoding reads the encoding flag
func setEncoding(cmd *cobra.Command, args []string) error {
	// validate and set encoding
	enc := viper.GetString("encoding")
	switch enc {
	case "hex":
		data.Encoder = data.HexEncoder
	case "b64":
		data.Encoder = data.B64Encoder
	case "btc":
		data.Encoder = base58.BTCEncoder
	default:
		return errors.Errorf("Unsupported encoding: %s", enc)
	}
	return nil
}

func validateOutput(cmd *cobra.Command, args []string) error {
	// validate output format
	output := viper.GetString(OutputFlag)
	switch output {
	case "text", "json":
	default:
		return errors.Errorf("Unsupported output format: %s", output)
	}
	return nil
}

/////////////////////////////
// Quick Flag Register
////////////////////////////

//Registering flags can be quickly achieved through using the utility functions
//RegisterFlags, and RegisterPersistentFlags. Ex:
//	flags := []Flag2Register{
//		{&myStringFlag, "mystringflag", "foobar", "description of what this flag does"},
//		{&myBoolFlag, "myboolflag", false, "description of what this flag does"},
//		{&myInt64Flag, "myintflag", 333, "description of what this flag does"},
//	}
//	RegisterFlags(MyCobraCmd, flags)
type Flag2Register struct {
	Pointer interface{}
	Use     string
	Value   interface{}
	Desc    string
}

//register flag utils
func RegisterFlags(c *cobra.Command, flags []Flag2Register) {
	registerFlags(c, flags, false)
}

func RegisterPersistentFlags(c *cobra.Command, flags []Flag2Register) {
	registerFlags(c, flags, true)
}

func registerFlags(c *cobra.Command, flags []Flag2Register, persistent bool) {

	var flagset *pflag.FlagSet
	if persistent {
		flagset = c.PersistentFlags()
	} else {
		flagset = c.Flags()
	}

	for _, f := range flags {

		ok := false

		switch f.Value.(type) {
		case string:
			if _, ok = f.Pointer.(*string); ok {
				flagset.StringVar(f.Pointer.(*string), f.Use, f.Value.(string), f.Desc)
			}
		case int:
			if _, ok = f.Pointer.(*int); ok {
				flagset.IntVar(f.Pointer.(*int), f.Use, f.Value.(int), f.Desc)
			}
		case uint64:
			if _, ok = f.Pointer.(*uint64); ok {
				flagset.Uint64Var(f.Pointer.(*uint64), f.Use, f.Value.(uint64), f.Desc)
			}
		case bool:
			if _, ok = f.Pointer.(*bool); ok {
				flagset.BoolVar(f.Pointer.(*bool), f.Use, f.Value.(bool), f.Desc)
			}
		}

		if !ok {
			panic("improper use of RegisterFlags")
		}
	}
}
