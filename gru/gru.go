package gru

import (
	"fmt"
	"math/rand"
	"sort"

	G "gorgonia.org/gorgonia"
)

// Chunks are SQL chunks
var Chunks = []string{
	"/*",
	"*/",
	"--",
	"begin",
	"end",
	"set",
	"select",
	"count",
	"top",
	"into",
	"as",
	"from",
	"where",
	"exists",
	"and",
	"&&",
	"or",
	"||",
	"not",
	"in",
	"like",
	"is",
	"between",
	"union",
	"all",
	"having",
	"order",
	"group",
	"by",
	"print",
	"var",
	"char",
	"master",
	"cmdshell",
	"waitfor",
	"delay",
	"time",
	"exec",
	"immediate",
	"declare",
	"sleep",
	"md5",
	"benchmark",
	"load",
	"file",
	"schema",
	"null",
	"version",
}

func init() {
	sort.Slice(Chunks, func(i, j int) bool {
		a, b := Chunks[i], Chunks[j]
		if la, lb := len(a), len(b); la > lb {
			return true
		} else if la == lb {
			return a < b
		}
		return false
	})
}

// GRU is a GRU based anomaly detection engine
type GRU struct {
	*Model
	learner   []*RNN
	inference *RNN
	solver    G.Solver
	steps     int
}

// NewGRU creates a new GRU anomaly detection engine
func NewGRU(rnd *rand.Rand) *GRU {
	steps := 3
	inputSize := 256 + len(Chunks)
	embeddingSize := 10
	outputSize := 2
	hiddenSizes := []int{5}
	gru := NewModel(rnd, 2, inputSize, embeddingSize, outputSize, hiddenSizes)

	learner := make([]*RNN, steps)
	for i := range learner {
		learner[i] = NewRNN(gru)
		err := learner[i].ModeLearn(i + 1)
		if err != nil {
			panic(err)
		}
	}

	inference := NewRNN(gru)
	err := inference.ModeInference()
	if err != nil {
		panic(err)
	}

	learnrate := 0.001
	l2reg := 0.000001
	clipVal := 5.0
	solver := G.NewRMSPropSolver(G.WithLearnRate(learnrate), G.WithL2Reg(l2reg), G.WithClip(clipVal))

	return &GRU{
		Model:     gru,
		learner:   learner,
		inference: inference,
		solver:    solver,
		steps:     steps,
	}
}

func convert(input []byte) []int {
	length, i := len(input), 0
	data := make([]int, 0, length)
conversion:
	for i < length {
	search:
		for j, v := range Chunks {
			chunk := []byte(v)
			for k, c := range chunk {
				index := i + k
				if index >= len(input) {
					continue search
				}
				if c != input[index] {
					continue search
				}
			}
			data = append(data, 256+j)
			i += len(chunk)
			continue conversion
		}
		data = append(data, int(input[i]))
		i++
	}

	return data
}

// Train trains the GRU
func (g *GRU) Train(input []byte, attack bool) float32 {
	data := convert(input)
	learner := g.learner[len(g.learner)-1]
	if len(data) < len(g.learner) {
		learner = g.learner[len(data)-1]
	}
	cost, _, err := learner.Learn(data, attack, g.solver)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}
	total := 0.0
	for _, v := range cost {
		total += v
	}

	return float32(total) / float32(len(cost))
}

// Test tests a string
func (g *GRU) Test(input []byte) bool {
	data := convert(input)
	return g.inference.IsAttack(data)
}
