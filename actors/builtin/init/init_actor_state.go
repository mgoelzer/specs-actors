package init

import (
	addr "github.com/filecoin-project/go-address"
	cid "github.com/ipfs/go-cid"
	errors "github.com/pkg/errors"
	cbg "github.com/whyrusleeping/cbor-gen"

	abi "github.com/filecoin-project/specs-actors/actors/abi"
	builtin "github.com/filecoin-project/specs-actors/actors/builtin"
	autil "github.com/filecoin-project/specs-actors/actors/util"
	adt "github.com/filecoin-project/specs-actors/actors/util/adt"
)

type AddrKey = adt.AddrKey

type State struct {
	AddressMap  cid.Cid // HAMT[addr.Address]abi.ActorID
	NextID      abi.ActorID
	NetworkName string
}

func ConstructState(addressMapRoot cid.Cid, networkName string) *State {
	return &State{
		AddressMap:  addressMapRoot,
		NextID:      abi.ActorID(builtin.FirstNonSingletonActorId),
		NetworkName: networkName,
	}
}

// ResolveAddress resolves an address to an ID-address, if possible.
// If the provided address is an ID address, it is returned as-is.
// This means that ID-addresses (which should only appear as values, not keys) and singleton actor addresses
// pass through unchanged.
//
// Post-condition: all addresses succesfully returned by this method satisfy `addr.Protocol() == addr.ID`.
func (s *State) ResolveAddress(store adt.Store, address addr.Address) (addr.Address, error) {
	// Short-circuit ID address resolution.
	if address.Protocol() == addr.ID {
		return address, nil
	}

	// Lookup address.
	m := adt.AsMap(store, s.AddressMap)
	var actorID cbg.CborInt
	found, err := m.Get(AddrKey(address), &actorID)
	if err != nil {
		return addr.Undef, errors.Wrapf(err, "resolve address failed to look up map")
	}
	if found {
		// Reconstruct address from the ActorID.
		idAddr, err2 := addr.NewIDAddress(uint64(actorID))
		autil.Assert(err2 == nil)
		return idAddr, nil
	}

	// Not found.
	return addr.Undef, errors.New("address not found")
}

// Allocates a new ID address and stores a mapping of the argument address to it.
// Returns the newly-allocated address.
func (s *State) MapAddressToNewID(store adt.Store, address addr.Address) (addr.Address, error) {
	actorID := cbg.CborInt(s.NextID)
	s.NextID++

	m := adt.AsMap(store, s.AddressMap)
	err := m.Put(AddrKey(address), &actorID)
	if err != nil {
		return addr.Undef, errors.Wrapf(err, "map address failed to store entry")
	}
	s.AddressMap = m.Root()

	idAddr, err := addr.NewIDAddress(uint64(actorID))
	autil.Assert(err == nil)
	return idAddr, nil
}
