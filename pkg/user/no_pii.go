package user

// NoPIIUser is used for tracking age range buckets and countries for users
// in our system.
type NoPIIUser struct {
	AgeRangeLow  int32  `json:"age_range_low"`
	AgeRangeHigh int32  `json:"age_range_high"`
	Country      string `json:"country"`
}
