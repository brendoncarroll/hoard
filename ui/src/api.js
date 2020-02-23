
const queryEndpoint = "/query"

function queryManifests(query) {
    if (query === undefined || query === null) {
        query = {limit: 10}
    }

    return fetch(queryEndpoint, {
        method: 'POST',
        body: JSON.stringify(query),
    }).then(res => {
        if (res.ok) {
            return res.json()
        } else {
            console.log(res)
            return Promise.reject("response not okay")
        }
    })
}

const suggestEndpoint = "/suggest"

function suggestTags(id) {
    let url = suggestEndpoint + "/" + id.toString()
    return fetch(url, {
        method: 'GET',
    }).then(res => {
        if (res.ok) {
            return res.json()
        } else {
            return Promise.reject("response not okay")
        }
    })
}

function makeDataURL(id) {
    return `/d/${id}`
}

export {
    queryManifests,
    suggestTags,
    makeDataURL,
}