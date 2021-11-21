import React from 'react';

import Channel from './channel.js';

class ChannelList extends React.Component {
  render() {
    const channels = [];
    for (const channel of this.props.channels) {
      channels.push(
        <li key={channel.id}>
          <Channel id={channel.id} name={channel.name} />
        </li>
      );
    }
    return (
      <div className="channels-list">
        <ul className="list-unstyled components">
          {channels}
        </ul>
      </div>
    );
  }
}

export default ChannelList;
