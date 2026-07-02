package keeper

type Keeper struct{}

func NewKeeper(
	cdc interface{},
	storeService interface{},
	channelKeeper interface{},
	ics4Wrapper interface{},
	portKeeper interface{},
	authKeeper interface{},
	bankKeeper interface{},
) Keeper {
	return Keeper{}
}
