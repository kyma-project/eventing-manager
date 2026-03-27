Summary
Implement Kubernetes Native RBAC Aggregation for the eventing module as per the Kyma RBAC Aggregation Decision Record.

Background
The Kyma project has adopted the Kubernetes Native RBAC Aggregation pattern to ensure that standard Kubernetes User-Facing Roles (admin, edit, view) automatically inherit permissions for Kyma-specific resources. This eliminates operational friction and ensures that users bound to standard roles can manage module resources without manual role configuration.

Requirements
Mandatory Roles
The eventing module must provide the following aggregated ClusterRoles:

1. View Role (kyma-eventing-view)
   Purpose: Read-only access to module resources
   Label: rbac.authorization.k8s.io/aggregate-to-view: "true"
   Allowed Verbs: get, list, watch only
   Restrictions: Must NOT include access to sensitive resources (e.g., Secrets)
2. Edit Role (kyma-eventing-edit)
   Purpose: Read/write access to module resources
   Label: rbac.authorization.k8s.io/aggregate-to-edit: "true"
   Allowed Verbs: get, list, watch, create, update, patch, delete, deletecollection
   Note: The admin role automatically aggregates all rules from edit, so this also extends admin
   Optional Roles
3. Admin Role (kyma-eventing-admin) - If Needed
   Purpose: Admin-only operations beyond edit permissions (e.g., cluster-wide deletions, security configurations)
   Label: rbac.authorization.k8s.io/aggregate-to-admin: "true"
   When to Create: Only if the module has specific admin-only operations not covered by edit permissions
4. Granular Atomic Roles - If Applicable
   Purpose: Narrower scopes for specific use cases (e.g., kyma-eventing-<resource>-developer)
   When to Create: When cluster administrators need more targeted permissions than the default roles provide
   ClusterRole Examples
   View Role Example
   apiVersion: rbac.authorization.k8s.io/v1
   kind: ClusterRole
   metadata:
   name: kyma-eventing-view
   labels:
   rbac.authorization.k8s.io/aggregate-to-view: "true"
   rules:
- apiGroups:
    - "<module-api-group>"
      resources:
    - "<resource-1>"
    - "<resource-2>"
      verbs:
    - get
    - list
    - watch
      Edit Role Example
      apiVersion: rbac.authorization.k8s.io/v1
      kind: ClusterRole
      metadata:
      name: kyma-eventing-edit
      labels:
      rbac.authorization.k8s.io/aggregate-to-edit: "true"
      rules:
- apiGroups:
    - "<module-api-group>"
      resources:
    - "<resource-1>"
    - "<resource-2>"
      verbs:
    - get
    - list
    - watch
    - create
    - update
    - patch
    - delete
    - deletecollection
      Acceptance Criteria

View ClusterRole Created: kyma-eventing-view ClusterRole is created with correct aggregation label

Edit ClusterRole Created: kyma-eventing-edit ClusterRole is created with correct aggregation label

View Role Restrictions: View role contains ONLY read verbs (get, list, watch)

No Sensitive Resources in View: View role does NOT grant access to Secrets or other sensitive resources

All Module CRDs Covered: All Custom Resource Definitions owned by the module are included in the roles

Roles Deployed with Module: ClusterRoles are included in the module's deployment manifests

Documentation Updated: Module documentation reflects the new RBAC roles

Release Notes: RBAC changes are documented in release notes
Testing
Deploy the module with the new ClusterRoles
Create a ServiceAccount bound to the standard view ClusterRole
Verify the ServiceAccount can get, list, watch module resources
Verify the ServiceAccount CANNOT create, update, delete module resources
Create a ServiceAccount bound to the standard edit ClusterRole
Verify the ServiceAccount can perform all CRUD operations on module resources
Verify the ServiceAccount bound to admin ClusterRole has the same permissions as edit (unless admin-specific role was created)