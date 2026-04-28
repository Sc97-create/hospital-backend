package utils

import (
	"fmt"
	"strconv"
)

//prefix p
// total floors 3
// room per floor 10
// starting per floor 1
// p101,p102,p103,p104,p105,p106,p107,p108,p109,p110
// p201,p202,p203,p204,p205,p206,p207,p208,p209,p210
// p301,p302,p303,p304,p305,p306,p307,p308,p309,p310
// if totalfloors == len(count) => room number generated

func GenerateRoomNumber(prefix string, totalfloors int, roomperfloor int, startingperfloor int) map[int][]string {
	roomnumbermap := map[int][]string{}
	for i := 1; i <= totalfloors; i++ {
		initialfloor := i * 100
		for j := startingperfloor; j <= roomperfloor; j++ {
			roomnumbermap[i] = append(roomnumbermap[i], prefix+strconv.Itoa(initialfloor+j))
		}
	}
	return roomnumbermap
}
func GenerateBeds(bedsperroom int, roomtypename string) []string {
	var beds []string
	for i := 1; i <= bedsperroom; i++ {
		//concate with room_type name
		bednumber := fmt.Sprintf("%s-B%02d", roomtypename[:2], i)
		beds = append(beds, bednumber)
	}
	return beds
}
func CalculateSkip(offset string, limit string) (int, int) {
	offsetInt, err := strconv.Atoi(offset)
	if err != nil {
		return 0, 0
	}
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		return 0, 0
	}
	var skip int
	if offsetInt != 0 {
		skip = (offsetInt - 1) * limitInt
	}
	return skip, limitInt
}
