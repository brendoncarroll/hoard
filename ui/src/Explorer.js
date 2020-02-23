import React from 'react';
import {queryManifests, suggestTags} from './api.js';
import {Paper, Grid, TextField} from '@material-ui/core';
import {ManifestMedium} from './Manifest.js';

class Explorer extends React.Component {
    constructor(props) {
        super(props);
        this.state = {manifests: []}
    }

    componentDidMount() {
    queryManifests().then(res => {
      this.setState({
        manifests: res.manifests,
      })

      res.manifests.forEach(mf => {
        suggestTags(mf.id).then(tags => {
          let i = this.state.manifests.findIndex(x => x.id === mf.id)
          let mfs = this.state.manifests
          mfs[i].suggestedTags = tags

          this.setState({
            manifests: mfs,
          })
        })
      })

    })
  }

    render() {
        return <Grid container spacing={2}>
        <Grid item container justify="center" xs={12}>
          <SearchBar/>
        </Grid>
        
        <Grid item container direction="column" spacing={2}>
          {this.state.manifests.map(mf => {
            return (<ManifestMedium key={mf.id} {...mf} />)
           })
          }
        </Grid>
        </Grid>
    }
}

const SearchBar = (props) => {
    return <Paper>
        <TextField id='search' label='Search' fullWidth={true}></TextField>
    </Paper>
}

export default Explorer;