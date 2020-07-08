import angular from 'angular';
import _ from 'lodash-es';

class CreateEdgeStackViewController {
  constructor($state, EdgeStackService, EdgeGroupService, EdgeTemplateService, Notifications, FormHelper, $async) {
    Object.assign(this, { $state, EdgeStackService, EdgeGroupService, EdgeTemplateService, Notifications, FormHelper, $async });

    this.formValues = {
      Name: '',
      StackFileContent: '',
      StackFile: null,
      RepositoryURL: '',
      RepositoryReferenceName: '',
      RepositoryAuthentication: false,
      RepositoryUsername: '',
      RepositoryPassword: '',
      Env: [],
      ComposeFilePathInRepository: 'docker-compose.yml',
      Groups: [],
    };

    this.state = {
      Method: 'editor',
      formValidationError: '',
      actionInProgress: false,
      StackType: null,
    };

    this.edgeGroups = null;

    this.createStack = this.createStack.bind(this);
    this.createStackAsync = this.createStackAsync.bind(this);
    this.validateForm = this.validateForm.bind(this);
    this.createStackByMethod = this.createStackByMethod.bind(this);
    this.createStackFromFileContent = this.createStackFromFileContent.bind(this);
    this.createStackFromFileUpload = this.createStackFromFileUpload.bind(this);
    this.createStackFromGitRepository = this.createStackFromGitRepository.bind(this);
    this.editorUpdate = this.editorUpdate.bind(this);
    this.onChangeTemplate = this.onChangeTemplate.bind(this);
    this.onChangeTemplateAsync = this.onChangeTemplateAsync.bind(this);
    this.onChangeMethod = this.onChangeMethod.bind(this);
  }

  async $onInit() {
    try {
      this.edgeGroups = await this.EdgeGroupService.groups();
      this.noGroups = this.edgeGroups.length === 0;
    } catch (err) {
      this.Notifications.error('Failure', err, 'Unable to retrieve Edge groups');
    }

    try {
      const templates = await this.EdgeTemplateService.edgeTemplates();
      this.templates = _.map(templates, (template) => ({ ...template, label: `${template.title} - ${template.description}` }));
    } catch (err) {
      this.Notifications.error('Failure', err, 'Unable to retrieve Templates');
    }
  }

  createStack() {
    return this.$async(this.createStackAsync);
  }

  onChangeMethod() {
    this.formValues.StackFileContent = '';
    this.selectedTemplate = null;
  }

  onChangeTemplate(template) {
    return this.$async(this.onChangeTemplateAsync, template);
  }

  async onChangeTemplateAsync(template) {
    this.formValues.StackFileContent = '';
    try {
      const fileContent = await this.EdgeTemplateService.edgeTemplate(template);
      this.formValues.StackFileContent = fileContent;
    } catch (err) {
      this.Notifications.error('Failure', err, 'Unable to retrieve Template');
    }
  }

  async createStackAsync() {
    const name = this.formValues.Name;
    let method = this.state.Method;

    if (method === 'template') {
      method = 'editor';
    }

    if (!this.validateForm(method)) {
      return;
    }

    this.state.actionInProgress = true;
    try {
      await this.createStackByMethod(name, method);

      this.Notifications.success('Stack successfully deployed');
      this.$state.go('edge.stacks');
    } catch (err) {
      this.Notifications.error('Deployment error', err, 'Unable to deploy stack');
    } finally {
      this.state.actionInProgress = false;
    }
  }

  validateForm(method) {
    this.state.formValidationError = '';

    if (method === 'editor' && this.formValues.StackFileContent === '') {
      this.state.formValidationError = 'Stack file content must not be empty';
      return;
    }

    return true;
  }

  createStackByMethod(name, method) {
    switch (method) {
      case 'editor':
        return this.createStackFromFileContent(name);
      case 'upload':
        return this.createStackFromFileUpload(name);
      case 'repository':
        return this.createStackFromGitRepository(name);
    }
  }

  createStackFromFileContent(name) {
    return this.EdgeStackService.createStackFromFileContent(name, this.formValues.StackFileContent, this.formValues.Groups);
  }

  createStackFromFileUpload(name) {
    return this.EdgeStackService.createStackFromFileUpload(name, this.formValues.StackFile, this.formValues.Groups);
  }

  createStackFromGitRepository(name) {
    const repositoryOptions = {
      RepositoryURL: this.formValues.RepositoryURL,
      RepositoryReferenceName: this.formValues.RepositoryReferenceName,
      ComposeFilePathInRepository: this.formValues.ComposeFilePathInRepository,
      RepositoryAuthentication: this.formValues.RepositoryAuthentication,
      RepositoryUsername: this.formValues.RepositoryUsername,
      RepositoryPassword: this.formValues.RepositoryPassword,
    };
    return this.EdgeStackService.createStackFromGitRepository(name, repositoryOptions, this.formValues.Groups);
  }

  editorUpdate(cm) {
    this.formValues.StackFileContent = cm.getValue();
  }
}

angular.module('portainer.edge').controller('CreateEdgeStackViewController', CreateEdgeStackViewController);
export default CreateEdgeStackViewController;
