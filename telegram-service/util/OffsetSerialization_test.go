package util

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/weoses/memelo/telegram-service/entity"
)

func TestParseOffset(t *testing.T) {
	type args struct {
		offset string
	}
	tests := []struct {
		name string
		args args
		want *entity.PaginationOffset
	}{{
		name: "simple",
		args: args{
			offset: "BAB0ZXN0AwAqAAAAAh3DDm0v90AKg7m4dkeBNLoBBAB0ZXN0",
		},
		want: &entity.PaginationOffset{
			Searcher: "test",
			SortingAfter: []string{
				"42",
				"1dc30e6d-2ff7-400a-83b9-b876478134ba",
				"test",
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseOffset(tt.args.offset); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseOffset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSerializeOffset(t *testing.T) {
	type args struct {
		p *entity.PaginationOffset
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "simple",
			args: args{
				p: &entity.PaginationOffset{
					Searcher: "test",
					SortingAfter: []string{
						"42",
						"1dc30e6d-2ff7-400a-83b9-b876478134ba",
						"test",
					},
				},
			},
			want: "BAB0ZXN0AwAqAAAAAh3DDm0v90AKg7m4dkeBNLoBBAB0ZXN0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SerializeOffset(tt.args.p); got != tt.want {
				t.Errorf("SerializeOffset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSerializeDeserialize(t *testing.T) {
	type args struct {
		p *entity.PaginationOffset
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "simple",
			args: args{
				p: &entity.PaginationOffset{
					Searcher: "test",
					SortingAfter: []string{
						"42",
						"1dc30e6d-2ff7-400a-83b9-b876478134ba",
						"test",
					},
				},
			},
		},
		{
			name: "realone",
			args: args{
				p: &entity.PaginationOffset{
					Searcher: "hybrid",
					SortingAfter: []string{
						"1776250313442464",
						"1dc30e6d-2ff7-400a-83b9-b876478134ba",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serialized := SerializeOffset(tt.args.p)
			deserialized := ParseOffset(serialized)
			if !cmp.Equal(deserialized, tt.args.p) {
				t.Errorf("SerializeOffset() + ParseOffset() = %v, want %v", deserialized, tt.args.p)
			}
			if len(serialized) > 64 {
				t.Errorf("Got too big token - %d = %v", len(serialized), serialized)
			}
		})
	}
}
