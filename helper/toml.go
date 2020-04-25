package helper

import (
	"bytes"
	"text/template"

	"github.com/spf13/viper"
	cmn "github.com/tendermint/tendermint/libs/common"
)

// Note: any changes to the comments/variables/mapstructure
// must be reflected in the appropriate struct in helper/config.go
const defaultConfigTemplate = `# This is a TOML config file.
# For more information, see https://github.com/toml-lang/toml

##### RPC configrations #####

# RPC endpoint for ethereum chain
eth_RPC_URL = "{{ .EthRPCUrl }}"

# RPC endpoint for bor chain
bor_RPC_URL = "{{ .BorRPCUrl }}"


# RPC endpoint for tendermint
tendermint_RPC_URL = "{{ .TendermintRPCUrl }}"


##### MQTT and Rest Server Config #####

# MQTT endpoint
amqp_url = "{{ .AmqpURL }}"

# Heimdall REST server endpoint
heimdall_rest_server = "{{ .HeimdallServerURL }}"


##### Intervals #####
child_chain_block_interval = "{{ .ChildBlockInterval }}"

## Bridge Poll Intervals
checkpoint_poll_interval = "{{ .CheckpointerPollInterval }}"
syncer_poll_interval = "{{ .SyncerPollInterval }}"
noack_poll_interval = "{{ .NoACKPollInterval }}"
clerk_polling_interval = "{{ .ClerkPollingInterval }}"
span_polling_interval = "{{ .SpanPollingInterval }}"

#### gas limits ####
main_chain_gas_limit = "{{ .MainchainGasLimit }}"

##### Timeout Config #####

no_ack_wait_time = "{{ .NoACKWaitTime }}"

`

var configTemplate *template.Template

func init() {
	var err error
	tmpl := template.New("appConfigFileTemplate")
	if configTemplate, err = tmpl.Parse(defaultConfigTemplate); err != nil {
		panic(err)
	}
}

// ParseConfig retrieves the default environment configuration for the
// application.
func ParseConfig() (*Configuration, error) {
	conf := GetDefaultHeimdallConfig()
	err := viper.Unmarshal(conf)
	return &conf, err
}

// WriteConfigFile renders config using the template and writes it to
// configFilePath.
func WriteConfigFile(configFilePath string, config *Configuration) {
	var buffer bytes.Buffer

	if err := configTemplate.Execute(&buffer, config); err != nil {
		panic(err)
	}

	cmn.MustWriteFile(configFilePath, buffer.Bytes(), 0644)
}
