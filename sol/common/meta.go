package common

import (
	"github.com/gagliardetto/solana-go"
	"github.com/near/borsh-go"
)

type Data struct {
	Name                 string
	Symbol               string
	Uri                  string
	SellerFeeBasisPoints uint16
	Creators             *[]Creator
}

type Creator struct {
	Address  solana.PublicKey
	Verified bool
	Share    uint8
}

func MetadataDeserialize(data []byte) (Metadata, error) {
	var metadata Metadata

	err := borsh.Deserialize(&metadata, data)
	return metadata, err
}

type Metadata struct {
	Key                 uint8
	UpdateAuthority     solana.PublicKey
	Mint                solana.PublicKey
	Data                Data
	PrimarySaleHappened bool
	IsMutable           bool
	EditionNonce        *uint8
	TokenStandard       *uint8
	Collection          *Collection
	Uses                *Uses
	CollectionDetails   *CollectionDetails
	ProgrammableConfig  *ProgrammableConfig
}

type Collection struct {
	Verified bool
	Key      solana.PublicKey
}

type Uses struct {
	UseMethod uint8
	Remaining uint64
	Total     uint64
}

type CollectionDetails struct {
	Enum borsh.Enum `borsh_enum:"true"`
	V1   CollectionDetailsV1
}

type CollectionDetailsV1 struct {
	Size uint64
}

type ProgrammableConfig struct {
	Enum borsh.Enum `borsh_enum:"true"`
	V1   ProgrammableConfigV1
}

type ProgrammableConfigV1 struct {
	RuleSet *solana.PublicKey
}
