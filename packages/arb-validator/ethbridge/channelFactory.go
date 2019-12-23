package ethbridge

import (
	"github.com/offchainlabs/arbitrum/packages/arb-validator/ethbridge/channelfactory"
	errors2 "github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/offchainlabs/arbitrum/packages/arb-util/protocol"
	"github.com/offchainlabs/arbitrum/packages/arb-util/value"
	"github.com/offchainlabs/arbitrum/packages/arb-validator/valmessage"
)

type ChannelFactory struct {
	contract *channelfactory.ChannelFactory
	client   *ethclient.Client
}

func NewChannelFactory(address common.Address, client *ethclient.Client) (*ChannelFactory, error) {
	vmCreatorContract, err := channelfactory.NewChannelFactory(address, client)
	if err != nil {
		return nil, errors2.Wrap(err, "Failed to connect to ChannelFactory")
	}
	return &ChannelFactory{vmCreatorContract, client}, nil
}

func (con *ChannelFactory) ParseChannelCreated(log *types.Log) (common.Address, error) {
	event, err := con.contract.ParseChannelCreated(*log)
	if err != nil {
		return common.Address{}, err
	}
	return event.VmAddress, nil
}

func (con *ChannelFactory) CreateChannel(
	auth *bind.TransactOpts,
	config *valmessage.VMConfiguration,
	vmState [32]byte,
) (common.Address, error) {
	var owner common.Address
	copy(owner[:], config.Owner.Value)
	var escrowCurrency common.Address
	copy(escrowCurrency[:], config.EscrowCurrency.Value)
	validatorKeys := make([]common.Address, 0, len(config.AssertKeys))
	for _, key := range config.AssertKeys {
		validatorKeys = append(validatorKeys, protocol.NewAddressFromBuf(key))
	}
	tx, err := con.contract.CreateChannel(
		auth,
		vmState,
		uint32(config.GracePeriod),
		config.MaxExecutionStepCount,
		value.NewBigIntFromBuf(config.EscrowRequired),
		owner,
		validatorKeys,
	)
	if err != nil {
		return common.Address{}, errors2.Wrap(err, "Failed to call to ChannelFactory.CreateChannel")
	}
	receipt, err := waitForReceipt(auth.Context, con.client, auth, tx, "CreateChannel")
	if err != nil {
		return common.Address{}, err
	}
	if len(receipt.Logs) != 1 {
		return common.Address{}, errors2.New("Wrong receipt count")
	}
	event, err := con.contract.ParseChannelCreated(*receipt.Logs[0])
	if err != nil {
		return common.Address{}, err
	}
	return event.VmAddress, nil
}
