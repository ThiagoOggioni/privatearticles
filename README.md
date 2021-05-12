# Install the chaincode
minifab install -n privatearticles -r true

# Modify the vars/privatearticles_collection_config.json with the following content

```
[
 {
    "name": "collectionarticles",
    "policy": "OR( 'org0examplecom.member', 'org1examplecom.member' )",
    "requiredPeerCount": 0,
    "maxPeerCount": 3,
    "blockToLive":1000000,
    "memberOnlyRead": true
 },
 {
    "name": "collectionArticlePrivateDetails",
    "policy": "OR( 'org0examplecom.member' )",
    "requiredPeerCount": 0,
    "maxPeerCount": 3,
    "blockToLive":3,
    "memberOnlyRead": true
 }
]
```
# Approve,commit,initialize the chaincode
    minifab approve,commit,initialize -p ''

# To init article
    ARTICLE=$( echo '{"name":"article1","color":"blue","size":35,"owner":"tom","price":99}' | base64 | tr -d \\n )
    minifab invoke -p '"initArticle"' -t '{"article":"'$ARTICLE'"}'

    ARTICLE=$( echo '{"name":"article2","color":"red","size":50,"owner":"tom","price":102}' | base64 | tr -d \\n )
    minifab invoke -p '"initArticle"' -t '{"article":"'$ARTICLE'"}'

    ARTICLE=$( echo '{"name":"article5","color":"blue","size":70,"owner":"tom","price":103}' | base64 | tr -d \\n )
    minifab invoke -p '"initArticle"' -t '{"article":"'$ARTICLE'"}'

# To transfer article
    ARTICLE_OWNER=$( echo '{"name":"article2","owner":"jerry"}' | base64 | tr -d \\n )
    minifab invoke -p '"transferArticle"' -t '{"article_owner":"'$ARTICLE_OWNER'"}'

# To query article
    minifab query -p '"readArticle","article4"' -t ''
    minifab query -p '"readArticlePrivateDetails","article4"' -t ''
    minifab query -p '"getArticlesByRange","article1","article4"' -t ''

# To delete article
    ARTICLE_ID=$( echo '{"name":"article1"}' | base64 | tr -d \\n )
    minifab invoke -p '"delete"' -t '{"article_delete":"'$ARTICLE_ID'"}'
