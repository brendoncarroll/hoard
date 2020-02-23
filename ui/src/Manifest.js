import React from 'react';
import {Card, CardContent, Chip, Grid, Typography} from '@material-ui/core';
import {makeDataURL} from './api.js';

const ManifestMedium = (props) => {
    let suggestedTags = props.suggestedTags || {};

    return <Card variant='outlined'>
    <CardContent>
        <Grid container direction="row" justify='space-between'>
        <Typography variant="body2" color="textSecondary"><b>#</b>{props.id.toString().padStart(4, '0')}</Typography>

        <Grid container item direction='column' xs={4}>
            <Typography>Tags</Typography>
            <TagSet tags={new Map(Object.entries(props.tags))}/> 
        </Grid>
        <Grid container item direction='column' xs={4}>
            <Typography>Suggested Tags</Typography>
            <TagSet tags={new Map(Object.entries(suggestedTags))}/>
        </Grid>
        </Grid>
        <a href={makeDataURL(props.id)}>download</a>
    </CardContent>
    </Card>
}

const TagSet = (props) => {
    let chips = [];
    props.tags.forEach((v,k) => {
        let x = <span><b>{k}:</b> {v}</span>
        chips.push(
            <Chip key={k} label={x} variant='outlined'/>
        )
    })
    chips.sort((a,b) => a.key < b.key)

    return <div>
        {chips}
    </div>
}

export {
    ManifestMedium,
}
