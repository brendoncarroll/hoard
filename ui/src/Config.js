import React from 'react';
import {Grid} from '@material-ui/core';

class Config extends React.Component {
    constructor(props) {
        super(props)
        this.setState({peers:[]})
    }

    render() {
        return <Grid container direction="column">
            Config
        </Grid>
    }
}

export default Config;
