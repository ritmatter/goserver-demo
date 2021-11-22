import React from 'react';

class UserStatus extends React.Component {
  render() {
    return (
      <div className="userStatus">
        <div>Logged in as {this.props.id}</div>
      </div>
    );
  }
}

export default UserStatus;
