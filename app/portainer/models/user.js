export function UserViewModel(data) {
  this.Id = data.Id;
  this.Username = data.Username;
  this.Role = data.Role;
  this.EndpointAuthorizations = data.EndpointAuthorizations;
  this.PortainerAuthorizations = data.PortainerAuthorizations;
  if (data.Role === 1) {
    this.RoleName = 'administrator';
  } else {
    this.RoleName = 'user';
  }
  this.AuthenticationMethod = data.AuthenticationMethod;
  this.Checked = false;
}
