package main

import (
	"github.com/gnewton/gomesh"
	"log"
	"sort"
	"strings"
)

const DESCRIPTOR = "descriptor"
const QUALIFIER = "qualifier"
const SUPPLEMENTAL = "supplemental"
const PHARMACOLOGICAL = "pharmacological"
const TREE = "tree"

var NOUNS = []string{
	DESCRIPTOR,
	QUALIFIER,
	SUPPLEMENTAL,
	TREE,
	PHARMACOLOGICAL,
}
var allNouns []gomesh.IdEntry

var descMap map[string]*gomesh.DescriptorRecord
var descMap2 map[string]*LocalDesc
var descSlice []*gomesh.IdEntry

var suppMap map[string]*gomesh.SupplementalRecord
var suppSlice []*gomesh.IdEntry

var qualMap map[string]*gomesh.QualifierRecord
var qualSlice []*gomesh.IdEntry

var pharmMap map[string]*gomesh.PharmacologicalAction
var pharmSlice []*gomesh.IdEntry

var root *gomesh.Node
var treeMap map[string]*gomesh.Node

type LocalDesc gomesh.DescriptorRecord

type Paging struct {
	Limit            int    `json:"limit"`
	Offset           int    `json:"offset"`
	Count            int    `json:"count"`
	NextPageUrl      string `json:"nextPageUrl,omitempty"`
	PrevioustPageUrl string `json:"previousPageUrl,omitempty"`
}

func (desc *LocalDesc) setQualifierUrls(baseUrl string) {
	if desc.AllowableQualifiersList.AllowableQualifier != nil {
		for i := 0; i < len(desc.AllowableQualifiersList.AllowableQualifier); i++ {
			qualifierReferredTo := desc.AllowableQualifiersList.AllowableQualifier[i].QualifierReferredTo
			qualifierReferredTo.Url = baseUrl + "/" + QUALIFIER + "/" + qualifierReferredTo.QualifierUI
		}
	}
}

func (desc *LocalDesc) setTreeNumberUrls(baseUrl string) {
	if desc.TreeNumberList.TreeNumber != nil {
		for i := 0; i < len(desc.TreeNumberList.TreeNumber); i++ {
			tn := &(desc.TreeNumberList.TreeNumber[i])
			tn.Url = baseUrl + "/" + TREE + "/" + tn.TreeNumber
		}
	}
}

func (desc *LocalDesc) setDescUrls(baseUrl string) {
	if desc.PharmacologicalActionList.PharmacologicalAction != nil {
		for i := 0; i < len(desc.PharmacologicalActionList.PharmacologicalAction); i++ {
			ref := &(desc.PharmacologicalActionList.PharmacologicalAction[i])
			ref.DescriptorReferredTo.Url = baseUrl + "/" + DESCRIPTOR + "/" + ref.DescriptorReferredTo.DescriptorUI
		}
	}

	if desc.SeeRelatedList.SeeRelatedDescriptor != nil {
		for i := 0; i < len(desc.SeeRelatedList.SeeRelatedDescriptor); i++ {
			ref := &(desc.SeeRelatedList.SeeRelatedDescriptor[i])
			ref.DescriptorReferredTo.Url = baseUrl + "/" + DESCRIPTOR + "/" + ref.DescriptorReferredTo.DescriptorUI
		}
	}
}

func loadData() error {
	treeMap = make(map[string]*gomesh.Node)
	var err error
	log.Println("Start Loading MeSH XML...")

	////////////////
	log.Println("\tLoading Supplemental MeSH XML from file: ", *SUPPLEMENTAL_XML_FILE)
	suppMap, err = gomesh.SupplementalMapFromFile(*SUPPLEMENTAL_XML_FILE)
	if err != nil {
		return err
	}
	index := 0
	suppSlice = make([]*gomesh.IdEntry, len(suppMap))
	for _, supp := range suppMap {
		newEntry := new(gomesh.IdEntry)
		newEntry.Id = supp.SupplementalRecordUI
		newEntry.Url = BASE_URL + "/" + SUPPLEMENTAL + "/" + newEntry.Id
		//newEntry.Url = SUPPLEMENTAL + "/" + newEntry.Id
		suppSlice[index] = newEntry
		index += 1

		for i := 0; i < len(supp.HeadingMappedToList.HeadingMappedTo); i++ {
			descriptorReferredTo := supp.HeadingMappedToList.HeadingMappedTo[i].DescriptorReferredTo
			descriptorReferredTo.Url = BASE_URL + "/" + DESCRIPTOR + "/" + strings.TrimLeft(descriptorReferredTo.DescriptorUI, "*")
		}
	}

	////////////////
	log.Println("\tLoading Pharmacological Action MeSH XML from file: ", *PHARMACOLOGICAL_XML_FILE)
	pharmMap, err = gomesh.PharmacologicalMapFromFile(*PHARMACOLOGICAL_XML_FILE)
	if err != nil {
		return err
	}
	index = 0
	pharmSlice = make([]*gomesh.IdEntry, len(pharmMap))
	for pharm := range pharmMap {
		newEntry := new(gomesh.IdEntry)
		newEntry.Id = pharmMap[pharm].DescriptorReferredTo.DescriptorUI
		newEntry.Url = BASE_URL + "/" + PHARMACOLOGICAL + "/" + newEntry.Id
		pharmSlice[index] = newEntry
		index += 1

		pharmMap[pharm].DescriptorReferredTo.Url = BASE_URL + "/" + DESCRIPTOR + "/" + pharm
		//if pharmMap[pharm].PharmacologicalActionSubstanceList.Substance != nil
		{
			for i := 0; i < len(pharmMap[pharm].PharmacologicalActionSubstanceList.Substance); i++ {
				//for _, substance := range pharmMap[pharm].PharmacologicalActionSubstanceList.Substance{
				substance := &pharmMap[pharm].PharmacologicalActionSubstanceList.Substance[i]
				if strings.Index(substance.RecordUI, "C") == 0 {
					substance.SupplementalUrl = BASE_URL + "/" + SUPPLEMENTAL + "/" + substance.RecordUI
				} else {
					substance.DescriptorUrl = BASE_URL + "/" + DESCRIPTOR + "/" + substance.RecordUI

				}
			}
		}
	}

	////////////////
	log.Println("\tLoading Qualifier MeSH XML from file:", *QUALIFIER_XML_FILE)
	qualMap, err = gomesh.QualifierMapFromFile(*QUALIFIER_XML_FILE)
	if err != nil {
		return err
	}
	qualSlice = make([]*gomesh.IdEntry, len(qualMap))
	index = 0
	for _, qual := range qualMap {
		newEntry := new(gomesh.IdEntry)
		newEntry.Id = qual.QualifierUI
		newEntry.Url = BASE_URL + "/" + QUALIFIER + "/" + newEntry.Id
		qualSlice[index] = newEntry
		index += 1
	}

	////////////////
	log.Println("\tLoading Descriptor MeSH XML from file: ", *DESCRIPTOR_XML_FILE)
	descMap, err = gomesh.DescriptorMapFromFile(*DESCRIPTOR_XML_FILE)
	if err != nil {
		return err
	}
	log.Println("\tBuilding name map")
	_ = gomesh.MeshDescriptorNameMap(descMap)

	descSlice = make([]*gomesh.IdEntry, len(descMap))
	index = 0
	descMap2 = make(map[string]*LocalDesc)

	for _, desc := range descMap {
		newEntry := new(gomesh.IdEntry)
		descriptorRecord := desc
		var localDesc = (*LocalDesc)(descriptorRecord)
		localDesc.setDescUrls(BASE_URL)
		localDesc.setTreeNumberUrls(BASE_URL)
		localDesc.setQualifierUrls(BASE_URL)

		descMap2[desc.DescriptorUI] = localDesc
		newEntry.Id = desc.DescriptorUI
		newEntry.Url = BASE_URL + "/" + DESCRIPTOR + "/" + newEntry.Id
		descSlice[index] = newEntry
		index += 1
	}

	sort.Sort(ById(descSlice))
	sort.Sort(ById(qualSlice))
	sort.Sort(ById(suppSlice))
	sort.Sort(ById(pharmSlice))

	root = gomesh.MakeTree(descMap)
	root.Traverse(0, AddUrlInfo)
	sort.Sort(ByIdX(root.Children))

	log.Println("Done Loading MeSH XML...")

	allNouns = make([]gomesh.IdEntry, len(NOUNS))
	for i, noun := range NOUNS {
		allNouns[i].Id = "/" + noun
		allNouns[i].Url = BASE_URL + "/" + noun
	}

	descMap = nil
	return nil
}

func AddUrlInfo(node *gomesh.Node) {
	//fmt.Println("AddUrlInfo", node.TreeNumber)
	treeMap[node.TreeNumber] = node
	if node.Children == nil {
		node.Children = make([]gomesh.IdEntry, len(node.ChildrenMap))
		if node.Descriptor != nil {
			node.DescriptorUrl = BASE_URL + "/" + DESCRIPTOR + "/" + node.Descriptor.DescriptorUI
		}
	}
	i := 0
	for _, childNode := range node.ChildrenMap {
		node.Children[i].Id = childNode.TreeNumber
		node.Children[i].Url = BASE_URL + "/" + TREE + "/" + childNode.TreeNumber
		node.Children[i].Label = childNode.Name
		i++
	}
}

//sort slices

type ByIdX []gomesh.IdEntry

type ById []*gomesh.IdEntry

func (a ById) Len() int            { return len(a) }
func (a ByIdX) Len() int           { return len(a) }
func (a ById) Swap(i, j int)       { a[i], a[j] = a[j], a[i] }
func (a ByIdX) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ById) Less(i, j int) bool  { return a[i].Id < a[j].Id }
func (a ByIdX) Less(i, j int) bool { return a[i].Id < a[j].Id }
