package browserforge

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

// BayesianNode implements a single node in a bayesian network
// allowing sampling from its conditional distribution.
type BayesianNode struct {
	name                     string
	parentNames              []string
	possibleValues           []string
	conditionalProbabilities map[string]interface{}
}

// NewBayesianNode creates a new BayesianNode from a node definition map.
func NewBayesianNode(nodeDef map[string]interface{}) *BayesianNode {
	node := &BayesianNode{}

	if name, ok := nodeDef["name"].(string); ok {
		node.name = name
	}

	if pn, ok := nodeDef["parentNames"].([]interface{}); ok {
		node.parentNames = make([]string, len(pn))
		for i, v := range pn {
			node.parentNames[i], _ = v.(string)
		}
	}

	if pv, ok := nodeDef["possibleValues"].([]interface{}); ok {
		node.possibleValues = make([]string, len(pv))
		for i, v := range pv {
			node.possibleValues[i], _ = v.(string)
		}
	}

	if cp, ok := nodeDef["conditionalProbabilities"].(map[string]interface{}); ok {
		node.conditionalProbabilities = cp
	}

	return node
}

// Name returns the node name.
func (n *BayesianNode) Name() string { return n.name }

// ParentNames returns the parent node names.
func (n *BayesianNode) ParentNames() []string { return n.parentNames }

// PossibleValues returns the possible values for this node.
func (n *BayesianNode) PossibleValues() []string { return n.possibleValues }

// GetProbabilitiesGivenKnownValues extracts unconditional probabilities of node values
// given the values of the parent nodes.
func (n *BayesianNode) GetProbabilitiesGivenKnownValues(parentValues map[string]interface{}) map[string]float64 {
	current := n.conditionalProbabilities

	for _, parentName := range n.parentNames {
		parentValue, exists := parentValues[parentName]
		if !exists {
			current = getMapOrEmpty(current, "skip")
			continue
		}

		parentValueStr, ok := parentValue.(string)
		if !ok {
			current = getMapOrEmpty(current, "skip")
			continue
		}

		if deeper, ok := current["deeper"].(map[string]interface{}); ok {
			if val, ok := deeper[parentValueStr].(map[string]interface{}); ok {
				current = val
				continue
			}
		}
		current = getMapOrEmpty(current, "skip")
	}

	// Convert leaf values to map[string]float64
	result := make(map[string]float64, len(current))
	for k, v := range current {
		if k == "deeper" || k == "skip" {
			continue
		}
		if f, ok := v.(float64); ok {
			result[k] = f
		}
	}
	return result
}

func getMapOrEmpty(m map[string]interface{}, key string) map[string]interface{} {
	if val, ok := m[key].(map[string]interface{}); ok {
		return val
	}
	return map[string]interface{}{}
}

// SampleRandomValueFromPossibilities randomly samples from the given values
// using the given probabilities.
func (n *BayesianNode) SampleRandomValueFromPossibilities(possibleValues []string, probabilities map[string]float64) string {
	anchor := rand.Float64()
	cumulativeProbability := 0.0
	for _, possibleValue := range possibleValues {
		cumulativeProbability += probabilities[possibleValue]
		if cumulativeProbability > anchor {
			return possibleValue
		}
	}
	// Default to first item
	if len(possibleValues) > 0 {
		return possibleValues[0]
	}
	return ""
}

// Sample randomly samples from the conditional distribution of this node
// given values of parents.
func (n *BayesianNode) Sample(parentValues map[string]interface{}) string {
	probabilities := n.GetProbabilitiesGivenKnownValues(parentValues)
	keys := make([]string, 0, len(probabilities))
	for k := range probabilities {
		keys = append(keys, k)
	}
	return n.SampleRandomValueFromPossibilities(keys, probabilities)
}

// SampleAccordingToRestrictions randomly samples from the conditional distribution
// of this node given restrictions on the possible values and the values of the parents.
// Returns the sampled value and true, or empty string and false if no valid value can be found.
func (n *BayesianNode) SampleAccordingToRestrictions(
	parentValues map[string]interface{},
	valuePossibilities []string,
	bannedValues []string,
) (string, bool) {
	probabilities := n.GetProbabilitiesGivenKnownValues(parentValues)

	bannedSet := make(map[string]bool, len(bannedValues))
	for _, v := range bannedValues {
		bannedSet[v] = true
	}

	var validValues []string
	for _, value := range valuePossibilities {
		if !bannedSet[value] {
			if _, ok := probabilities[value]; ok {
				validValues = append(validValues, value)
			}
		}
	}

	if len(validValues) > 0 {
		return n.SampleRandomValueFromPossibilities(validValues, probabilities), true
	}
	return "", false
}

// BayesianNetwork implements a bayesian network capable of randomly sampling
// from its distribution.
type BayesianNetwork struct {
	NodesInSamplingOrder []*BayesianNode
	NodesByName          map[string]*BayesianNode
}

// NewBayesianNetwork creates a new BayesianNetwork from a JSON or ZIP file path.
func NewBayesianNetwork(path string) (*BayesianNetwork, error) {
	networkDef, err := ExtractJSON(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load network: %w", err)
	}

	bn := &BayesianNetwork{
		NodesByName: make(map[string]*BayesianNode),
	}

	nodesRaw, ok := networkDef["nodes"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid network definition: missing 'nodes'")
	}

	bn.NodesInSamplingOrder = make([]*BayesianNode, len(nodesRaw))
	for i, nodeRaw := range nodesRaw {
		nodeDef, ok := nodeRaw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid node definition at index %d", i)
		}
		node := NewBayesianNode(nodeDef)
		bn.NodesInSamplingOrder[i] = node
		bn.NodesByName[node.Name()] = node
	}

	return bn, nil
}

// GenerateSample randomly samples from the distribution represented by the bayesian network.
func (bn *BayesianNetwork) GenerateSample(inputValues map[string]interface{}) map[string]interface{} {
	sample := make(map[string]interface{})
	if inputValues != nil {
		for k, v := range inputValues {
			sample[k] = v
		}
	}
	for _, node := range bn.NodesInSamplingOrder {
		if _, exists := sample[node.Name()]; !exists {
			sample[node.Name()] = node.Sample(sample)
		}
	}
	return sample
}

// GenerateConsistentSampleWhenPossible randomly samples values from the distribution,
// making sure the sample is consistent with the provided restrictions on value possibilities.
// Returns nil if no such sample can be generated.
func (bn *BayesianNetwork) GenerateConsistentSampleWhenPossible(valuePossibilities map[string][]string) map[string]interface{} {
	return bn.recursivelyGenerateConsistentSampleWhenPossible(
		make(map[string]interface{}), valuePossibilities, 0,
	)
}

func (bn *BayesianNetwork) recursivelyGenerateConsistentSampleWhenPossible(
	sampleSoFar map[string]interface{},
	valuePossibilities map[string][]string,
	depth int,
) map[string]interface{} {
	if depth == len(bn.NodesInSamplingOrder) {
		return sampleSoFar
	}

	node := bn.NodesInSamplingOrder[depth]
	var bannedValues []string

	possibilities := valuePossibilities[node.Name()]
	if possibilities == nil {
		possibilities = node.PossibleValues()
	}

	for {
		sampleValue, ok := node.SampleAccordingToRestrictions(sampleSoFar, possibilities, bannedValues)
		if !ok {
			break
		}

		sampleSoFar[node.Name()] = sampleValue
		nextSample := bn.recursivelyGenerateConsistentSampleWhenPossible(
			sampleSoFar, valuePossibilities, depth+1,
		)
		if nextSample != nil {
			return nextSample
		}

		bannedValues = append(bannedValues, sampleValue)
		delete(sampleSoFar, node.Name())
	}

	return nil
}

// ArrayIntersection performs a set "intersection" on the given arrays.
func ArrayIntersection(a, b []string) []string {
	setB := make(map[string]bool, len(b))
	for _, v := range b {
		setB[v] = true
	}
	var result []string
	for _, v := range a {
		if setB[v] {
			result = append(result, v)
		}
	}
	return result
}

// ArrayZip combines two arrays of string slices using the set union.
func ArrayZip(a, b [][]string) [][]string {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	result := make([][]string, minLen)
	for i := 0; i < minLen; i++ {
		seen := make(map[string]bool)
		var merged []string
		for _, v := range a[i] {
			if !seen[v] {
				seen[v] = true
				merged = append(merged, v)
			}
		}
		for _, v := range b[i] {
			if !seen[v] {
				seen[v] = true
				merged = append(merged, v)
			}
		}
		result[i] = merged
	}
	return result
}

// Undeeper removes the "deeper/skip" structures from the conditional probability table.
func Undeeper(obj interface{}) interface{} {
	m, ok := obj.(map[string]interface{})
	if !ok {
		return obj
	}

	result := make(map[string]interface{})
	for key, value := range m {
		if key == "skip" {
			continue
		}
		if key == "deeper" {
			if undeepened, ok := Undeeper(value).(map[string]interface{}); ok {
				for k, v := range undeepened {
					result[k] = v
				}
			}
		} else {
			result[key] = Undeeper(value)
		}
	}
	return result
}

// FilterByLastLevelKeys performs DFS on the tree and returns values of the nodes
// on the paths that end with the given keys (stored by levels).
func FilterByLastLevelKeys(tree map[string]interface{}, validKeys []string) [][]string {
	validKeySet := make(map[string]bool, len(validKeys))
	for _, k := range validKeys {
		validKeySet[k] = true
	}

	var out [][]string

	var recurse func(t map[string]interface{}, acc []string)
	recurse = func(t map[string]interface{}, acc []string) {
		for key, val := range t {
			valMap, isDict := val.(map[string]interface{})
			if !isDict || valMap == nil {
				if validKeySet[key] {
					accAsLevels := make([][]string, len(acc))
					for i, v := range acc {
						accAsLevels[i] = []string{v}
					}
					if len(out) == 0 {
						out = accAsLevels
					} else {
						out = ArrayZip(out, accAsLevels)
					}
				}
			} else {
				newAcc := make([]string, len(acc)+1)
				copy(newAcc, acc)
				newAcc[len(acc)] = key
				recurse(valMap, newAcc)
			}
		}
	}

	recurse(tree, nil)
	return out
}

// GetPossibleValues given a BayesianNetwork and a set of user constraints,
// returns an extended set of constraints induced by the original constraints
// and network structure.
func GetPossibleValues(network *BayesianNetwork, possibleValues map[string][]string) (map[string][]string, error) {
	var sets []map[string][]string

	for key, value := range possibleValues {
		if len(value) == 0 {
			return nil, fmt.Errorf(
				"the current constraints are too restrictive. No possible values can be found for the given constraints",
			)
		}

		node, ok := network.NodesByName[key]
		if !ok {
			continue
		}

		tree, ok := Undeeper(node.conditionalProbabilities).(map[string]interface{})
		if !ok {
			continue
		}

		zippedValues := FilterByLastLevelKeys(tree, value)

		setDict := make(map[string][]string)
		parentNames := node.ParentNames()
		for i, pName := range parentNames {
			if i < len(zippedValues) {
				setDict[pName] = zippedValues[i]
			}
		}
		setDict[key] = value
		sets = append(sets, setDict)
	}

	result := make(map[string][]string)
	for _, setDict := range sets {
		for key, values := range setDict {
			if existing, ok := result[key]; ok {
				intersected := ArrayIntersection(values, existing)
				if len(intersected) == 0 {
					return nil, fmt.Errorf(
						"the current constraints are too restrictive. No possible values can be found for the given constraints",
					)
				}
				result[key] = intersected
			} else {
				result[key] = values
			}
		}
	}

	return result, nil
}

// ExtractJSON unzips a zip file if the path points to a zip file,
// otherwise directly loads a JSON file.
func ExtractJSON(path string) (map[string]interface{}, error) {
	if filepath.Ext(path) != ".zip" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, err
		}
		return result, nil
	}

	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.HasSuffix(f.Name, ".json") {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			data, err := io.ReadAll(rc)
			if err != nil {
				return nil, err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(data, &result); err != nil {
				return nil, err
			}
			return result, nil
		}
	}

	return map[string]interface{}{}, nil
}
