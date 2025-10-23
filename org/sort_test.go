package org

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	pstore_client "github.com/brotherlogic/pstore/client"
	"google.golang.org/protobuf/proto"
)

var sortTable = []struct {
	file     string
	stype    pb.Sort
	expected []int64
}{
	{
		"mpl",
		pb.Sort_LABEL_CATNO,
		[]int64{
			915126,
			916092,
			920742,
			873459,
			997215,
			925701,
			927135,
			998295,
			927610,
			930912,
			931921,
			2956140,
			932974,
			1000394,
			936481,
			1002085,
			936519,
			937436,
			938804,
			2076270,
			939535,
			2908943,
			871908,
		},
	},
}

func TestOrdering(t *testing.T) {
	o := GetOrg(db.NewTestDB(pstore_client.GetTestClient()))

	for _, table := range sortTable {
		rs := &pb.RecordSet{}
		contentBytes, err := os.ReadFile(fmt.Sprintf("testdata/%v.hex", table.file))
		if err != nil {
			t.Fatalf("Error reading file: %v\n", err)
		}
		byteArray, err := hex.DecodeString(string(contentBytes))

		err = proto.Unmarshal(byteArray, rs)
		if err != nil {
			t.Fatalf("Unable to unmarshal file: %v", err)
		}

		org := &pb.Organisation{}
		weights := []*pb.LabelWeight{}

		switch table.stype {
		case pb.Sort_LABEL_CATNO:
			sort.SliceStable(rs.Records, func(i, j int) bool {
				return o.getLabelCatno(context.Background(), rs.Records[i], org, weights) < o.getLabelCatno(context.Background(), rs.Records[j], org, weights)
			})

			indexMap := make(map[int64]int)
			for i, r := range rs.GetRecords() {
				indexMap[r.GetRelease().GetId()] = i
			}

			for i := 0; i < len(table.expected)-1; i++ {
				if indexMap[table.expected[i]] > indexMap[table.expected[i+1]] {
					t.Fatalf("Sort mismatch %v -> %v is expected but %v -> %v was found", table.expected[i], table.expected[i+1], indexMap[table.expected[i]], indexMap[table.expected[i+1]])
				}
			}
		}
	}
}
