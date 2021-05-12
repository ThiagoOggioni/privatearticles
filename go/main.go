/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
)

// ArticlesPrivateChaincode example Chaincode implementation
type ArticlesPrivateChaincode struct {
}

type article struct {
	ObjectType string `json:"docType"` //docType is used to distinguish the various types of objects in state database
	Name       string `json:"name"`    //the fieldtags are needed to keep case from bouncing around
	Color      string `json:"color"`
	Size       int    `json:"size"`
	Owner      string `json:"owner"`
}

type articlePrivateDetails struct {
	ObjectType string `json:"docType"` //docType is used to distinguish the various types of objects in state database
	Name       string `json:"name"`    //the fieldtags are needed to keep case from bouncing around
	Price      int    `json:"price"`
}

// Init initializes chaincode
// ===========================
func (t *ArticlesPrivateChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

// Invoke - Our entry point for Invocations
// ========================================
func (t *ArticlesPrivateChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	fmt.Println("invoke is running " + function)

	// Handle different functions
	switch function {
	case "initArticle":
		//create a new article
		return t.initArticle(stub, args)
	case "readArticle":
		//read a article
		return t.readArticle(stub, args)
	case "readArticlePrivateDetails":
		//read a article private details
		return t.readArticlePrivateDetails(stub, args)
	case "transferArticle":
		//change owner of a specific article
		return t.transferArticle(stub, args)
	case "delete":
		//delete a article
		return t.delete(stub, args)
	case "getArticlesByRange":
		//get articles based on range query
		return t.getArticlesByRange(stub, args)
	case "getArticleHash":
		// get private data hash for collectionArticles
		return t.getArticleHash(stub, args)
	case "getArticlePrivateDetailsHash":
		// get private data hash for collectionArticlePrivateDetails
		return t.getArticlePrivateDetailsHash(stub, args)
	default:
		//error
		fmt.Println("invoke did not find func: " + function)
		return shim.Error("Received unknown function invocation")
	}
}

// ============================================================
// initArticle - create a new article, store into chaincode state
// ============================================================
func (t *ArticlesPrivateChaincode) initArticle(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error

	type articleTransientInput struct {
		Name  string `json:"name"` //the fieldtags are needed to keep case from bouncing around
		Color string `json:"color"`
		Size  int    `json:"size"`
		Owner string `json:"owner"`
		Price int    `json:"price"`
	}

	// ==== Input sanitation ====
	fmt.Println("- start init article")

	if len(args) != 0 {
		return shim.Error("Incorrect number of arguments. Private article data must be passed in transient map.")
	}

	transMap, err := stub.GetTransient()
	if err != nil {
		return shim.Error("Error getting transient: " + err.Error())
	}

	articleJsonBytes, ok := transMap["article"]
	if !ok {
		return shim.Error("article must be a key in the transient map")
	}

	if len(articleJsonBytes) == 0 {
		return shim.Error("article value in the transient map must be a non-empty JSON string")
	}

	var articleInput articleTransientInput
	err = json.Unmarshal(articleJsonBytes, &articleInput)
	if err != nil {
		return shim.Error("Failed to decode JSON of: " + string(articleJsonBytes))
	}

	if len(articleInput.Name) == 0 {
		return shim.Error("name field must be a non-empty string")
	}
	if len(articleInput.Color) == 0 {
		return shim.Error("color field must be a non-empty string")
	}
	if articleInput.Size <= 0 {
		return shim.Error("size field must be a positive integer")
	}
	if len(articleInput.Owner) == 0 {
		return shim.Error("owner field must be a non-empty string")
	}
	if articleInput.Price <= 0 {
		return shim.Error("price field must be a positive integer")
	}

	// ==== Check if article already exists ====
	articleAsBytes, err := stub.GetPrivateData("collectionArticles", articleInput.Name)
	if err != nil {
		return shim.Error("Failed to get article: " + err.Error())
	} else if articleAsBytes != nil {
		fmt.Println("This article already exists: " + articleInput.Name)
		return shim.Error("This article already exists: " + articleInput.Name)
	}

	// ==== Create article object and marshal to JSON ====
	article := &article{
		ObjectType: "article",
		Name:       articleInput.Name,
		Color:      articleInput.Color,
		Size:       articleInput.Size,
		Owner:      articleInput.Owner,
	}
	articleJSONasBytes, err := json.Marshal(article)
	if err != nil {
		return shim.Error(err.Error())
	}

	// === Save article to state ===
	err = stub.PutPrivateData("collectionArticles", articleInput.Name, articleJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	// ==== Create article private details object with price, marshal to JSON, and save to state ====
	articlePrivateDetails := &articlePrivateDetails{
		ObjectType: "articlePrivateDetails",
		Name:       articleInput.Name,
		Price:      articleInput.Price,
	}
	articlePrivateDetailsBytes, err := json.Marshal(articlePrivateDetails)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.PutPrivateData("collectionArticlePrivateDetails", articleInput.Name, articlePrivateDetailsBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	//  ==== Index the article to enable color-based range queries, e.g. return all blue articles ====
	//  An 'index' is a normal key/value entry in state.
	//  The key is a composite key, with the elements that you want to range query on listed first.
	//  In our case, the composite key is based on indexName~color~name.
	//  This will enable very efficient state range queries based on composite keys matching indexName~color~*
	indexName := "color~name"
	colorNameIndexKey, err := stub.CreateCompositeKey(indexName, []string{article.Color, article.Name})
	if err != nil {
		return shim.Error(err.Error())
	}
	//  Save index entry to state. Only the key name is needed, no need to store a duplicate copy of the article.
	//  Note - passing a 'nil' value will effectively delete the key from state, therefore we pass null character as value
	value := []byte{0x00}
	stub.PutPrivateData("collectionArticles", colorNameIndexKey, value)

	// ==== Article saved and indexed. Return success ====
	fmt.Println("- end init article")
	return shim.Success(nil)
}

// ===============================================
// readArticle - read a article from chaincode state
// ===============================================
func (t *ArticlesPrivateChaincode) readArticle(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var name, jsonResp string
	var err error

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting name of the article to query")
	}

	name = args[0]
	valAsbytes, err := stub.GetPrivateData("collectionArticles", name) //get the article from chaincode state
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + name + ": " + err.Error() + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"Article does not exist: " + name + "\"}"
		return shim.Error(jsonResp)
	}

	return shim.Success(valAsbytes)
}

// ===============================================
// readArticlereadArticlePrivateDetails - read a article private details from chaincode state
// ===============================================
func (t *ArticlesPrivateChaincode) readArticlePrivateDetails(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var name, jsonResp string
	var err error

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting name of the article to query")
	}

	name = args[0]
	valAsbytes, err := stub.GetPrivateData("collectionArticlePrivateDetails", name) //get the article private details from chaincode state
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get private details for " + name + ": " + err.Error() + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"Article private details does not exist: " + name + "\"}"
		return shim.Error(jsonResp)
	}

	return shim.Success(valAsbytes)
}

// ===============================================
// getArticleHash - get article private data hash for collectionArticles from chaincode state
// ===============================================
func (t *ArticlesPrivateChaincode) getArticleHash(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var name, jsonResp string
	var err error

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting name of the article to query")
	}

	name = args[0]
	valAsbytes, err := stub.GetPrivateDataHash("collectionArticles", name)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get article private data hash for " + name + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"Article private article data hash does not exist: " + name + "\"}"
		return shim.Error(jsonResp)
	}

	return shim.Success(valAsbytes)
}

// ===============================================
// getArticlePrivateDetailsHash - get article private data hash for collectionArticlePrivateDetails from chaincode state
// ===============================================
func (t *ArticlesPrivateChaincode) getArticlePrivateDetailsHash(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var name, jsonResp string
	var err error

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting name of the article to query")
	}

	name = args[0]
	valAsbytes, err := stub.GetPrivateDataHash("collectionArticlePrivateDetails", name)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get article private details hash for " + name + ": " + err.Error() + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"Article private details hash does not exist: " + name + "\"}"
		return shim.Error(jsonResp)
	}

	return shim.Success(valAsbytes)
}

// ==================================================
// delete - remove a article key/value pair from state
// ==================================================
func (t *ArticlesPrivateChaincode) delete(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	fmt.Println("- start delete article")

	type articleDeleteTransientInput struct {
		Name string `json:"name"`
	}

	if len(args) != 0 {
		return shim.Error("Incorrect number of arguments. Private article name must be passed in transient map.")
	}

	transMap, err := stub.GetTransient()
	if err != nil {
		return shim.Error("Error getting transient: " + err.Error())
	}

	articleDeleteJsonBytes, ok := transMap["article_delete"]
	if !ok {
		return shim.Error("article_delete must be a key in the transient map")
	}

	if len(articleDeleteJsonBytes) == 0 {
		return shim.Error("article_delete value in the transient map must be a non-empty JSON string")
	}

	var articleDeleteInput articleDeleteTransientInput
	err = json.Unmarshal(articleDeleteJsonBytes, &articleDeleteInput)
	if err != nil {
		return shim.Error("Failed to decode JSON of: " + string(articleDeleteJsonBytes))
	}

	if len(articleDeleteInput.Name) == 0 {
		return shim.Error("name field must be a non-empty string")
	}

	// to maintain the color~name index, we need to read the article first and get its color
	valAsbytes, err := stub.GetPrivateData("collectionArticles", articleDeleteInput.Name) //get the article from chaincode state
	if err != nil {
		return shim.Error("Failed to get state for " + articleDeleteInput.Name)
	} else if valAsbytes == nil {
		return shim.Error("Article does not exist: " + articleDeleteInput.Name)
	}

	var articleToDelete article
	err = json.Unmarshal([]byte(valAsbytes), &articleToDelete)
	if err != nil {
		return shim.Error("Failed to decode JSON of: " + string(valAsbytes))
	}

	// delete the article from state
	err = stub.DelPrivateData("collectionArticles", articleDeleteInput.Name)
	if err != nil {
		return shim.Error("Failed to delete state:" + err.Error())
	}

	// Also delete the article from the color~name index
	indexName := "color~name"
	colorNameIndexKey, err := stub.CreateCompositeKey(indexName, []string{articleToDelete.Color, articleToDelete.Name})
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.DelPrivateData("collectionArticles", colorNameIndexKey)
	if err != nil {
		return shim.Error("Failed to delete state:" + err.Error())
	}

	// Finally, delete private details of article
	err = stub.DelPrivateData("collectionArticlePrivateDetails", articleDeleteInput.Name)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

// ===========================================================
// transfer a article by setting a new owner name on the article
// ===========================================================
func (t *ArticlesPrivateChaincode) transferArticle(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	fmt.Println("- start transfer article")

	type articleTransferTransientInput struct {
		Name  string `json:"name"`
		Owner string `json:"owner"`
	}

	if len(args) != 0 {
		return shim.Error("Incorrect number of arguments. Private article data must be passed in transient map.")
	}

	transMap, err := stub.GetTransient()
	if err != nil {
		return shim.Error("Error getting transient: " + err.Error())
	}

	articleOwnerJsonBytes, ok := transMap["article_owner"]
	if !ok {
		return shim.Error("article_owner must be a key in the transient map")
	}

	if len(articleOwnerJsonBytes) == 0 {
		return shim.Error("article_owner value in the transient map must be a non-empty JSON string")
	}

	var articleTransferInput articleTransferTransientInput
	err = json.Unmarshal(articleOwnerJsonBytes, &articleTransferInput)
	if err != nil {
		return shim.Error("Failed to decode JSON of: " + string(articleOwnerJsonBytes))
	}

	if len(articleTransferInput.Name) == 0 {
		return shim.Error("name field must be a non-empty string")
	}
	if len(articleTransferInput.Owner) == 0 {
		return shim.Error("owner field must be a non-empty string")
	}

	articleAsBytes, err := stub.GetPrivateData("collectionArticles", articleTransferInput.Name)
	if err != nil {
		return shim.Error("Failed to get article:" + err.Error())
	} else if articleAsBytes == nil {
		return shim.Error("Article does not exist: " + articleTransferInput.Name)
	}

	articleToTransfer := article{}
	err = json.Unmarshal(articleAsBytes, &articleToTransfer) //unmarshal it aka JSON.parse()
	if err != nil {
		return shim.Error(err.Error())
	}
	articleToTransfer.Owner = articleTransferInput.Owner //change the owner

	articleJSONasBytes, _ := json.Marshal(articleToTransfer)
	err = stub.PutPrivateData("collectionArticles", articleToTransfer.Name, articleJSONasBytes) //rewrite the article
	if err != nil {
		return shim.Error(err.Error())
	}

	fmt.Println("- end transferArticle (success)")
	return shim.Success(nil)
}

// ===========================================================================================
// getArticlesByRange performs a range query based on the start and end keys provided.

// Read-only function results are not typically submitted to ordering. If the read-only
// results are submitted to ordering, or if the query is used in an update transaction
// and submitted to ordering, then the committing peers will re-execute to guarantee that
// result sets are stable between endorsement time and commit time. The transaction is
// invalidated by the committing peers if the result set has changed between endorsement
// time and commit time.
// Therefore, range queries are a safe option for performing update transactions based on query results.
// ===========================================================================================
func (t *ArticlesPrivateChaincode) getArticlesByRange(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	if len(args) < 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	startKey := args[0]
	endKey := args[1]

	resultsIterator, err := stub.GetPrivateDataByRange("collectionArticles", startKey, endKey)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten {
			buffer.WriteString(",")
		}

		buffer.WriteString(
			fmt.Sprintf(
				`{"Key":"%s", "Record":%s}`,
				queryResponse.Key, queryResponse.Value,
			),
		)
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	fmt.Printf("- getArticlesByRange queryResult:\n%s\n", buffer.String())

	return shim.Success(buffer.Bytes())
}

func main() {
	err := shim.Start(&ArticlesPrivateChaincode{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Exiting Simple chaincode: %s", err)
		os.Exit(2)
	}
}
