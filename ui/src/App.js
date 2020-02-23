import React from 'react';
import './App.css';
import {Tab, Typography, Box,Container, AppBar, Tabs} from '@material-ui/core';
import Explorer from './Explorer.js';
import Config from './Config.js';

class App extends React.Component {

  constructor(props) {
    super(props);
    this.state = {currentTab: 0};
  }

  tabChange(event, newValue) {
    this.setState({
      currentTab: newValue
    })
  }

  render() {
  return (
    <div className="App">
      <Container>
        <AppBar position="static">
        <Tabs value={this.state.currentTab} onChange={this.tabChange.bind(this)}>
          <Tab label="Explorer" index={0}/>
          <Tab label="Config" index={1}/>
        </Tabs>
        </AppBar>
        <TabPanel value={this.state.currentTab} index={0}>
          <Explorer/>
        </TabPanel>
        <TabPanel value={this.state.currentTab} index={1}>
          <Config/>
        </TabPanel>
      </Container>
    </div>
  );
  }
}

function TabPanel(props) {
  const { children, value, index, ...other } = props;

  return (
    <Typography
      component="div"
      role="tabpanel"
      hidden={value !== index}
      id={`simple-tabpanel-${index}`}
      aria-labelledby={`simple-tab-${index}`}
      {...other}
    >
      {value === index && <Box p={3}>{children}</Box>}
    </Typography>
  );
}

export default App;
