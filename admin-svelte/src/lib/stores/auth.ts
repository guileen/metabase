import { writable, derived } from 'svelte/store';
import { browser } from '$app/environment';

// Types
export interface User {
  id: string;
  username: string;
  email: string;
  role: 'admin' | 'user' | 'viewer';
  tenant_id: string;
  permissions: string[];
  created_at: string;
  last_login?: string;
  avatar?: string;
}

export interface Tenant {
  id: string;
  name: string;
  slug: string;
  domain?: string;
  settings: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface AuthState {
  user: User | null;
  tenant: Tenant | null;
  token: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
}

// Initial state
function createAuthStore() {
  const { subscribe, set, update } = writable<AuthState>({
    user: null,
    tenant: null,
    token: null,
    isAuthenticated: false,
    isLoading: false,
    error: null,
  });

  // Load from localStorage on client-side
  function loadFromStorage() {
    if (!browser) return;

    try {
      const stored = localStorage.getItem('metabase_auth');
      if (stored) {
        const parsed = JSON.parse(stored);
        update(state => ({
          ...state,
          ...parsed,
          isAuthenticated: !!(parsed.token && parsed.user),
        }));
      }
    } catch (error) {
      console.error('Failed to load auth from storage:', error);
    }
  }

  // Save to localStorage
  function saveToStorage(state: AuthState) {
    if (!browser) return;

    try {
      const toStore = {
        user: state.user,
        tenant: state.tenant,
        token: state.token,
      };
      localStorage.setItem('metabase_auth', JSON.stringify(toStore));
    } catch (error) {
      console.error('Failed to save auth to storage:', error);
    }
  }

  // Login function
  async function login(credentials: { username: string; password: string }) {
    update(state => ({
      ...state,
      isLoading: true,
      error: null,
    }));

    try {
      // In a real implementation, this would call the API
      // const response = await apiClient.post('/auth/login', credentials);

      // Mock response for development
      const mockResponse = {
        success: true,
        data: {
          user: {
            id: 'user_1',
            username: credentials.username,
            email: `${credentials.username}@example.com`,
            role: 'admin' as const,
            tenant_id: 'tenant_1',
            permissions: ['admin', 'read', 'write'],
            created_at: new Date().toISOString(),
            last_login: new Date().toISOString(),
          },
          tenant: {
            id: 'tenant_1',
            name: 'Default Organization',
            slug: 'default',
            settings: {},
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
          },
          token: 'mock_jwt_token_' + Date.now(),
        },
      };

      const { user, tenant, token } = mockResponse.data;

      const newState: AuthState = {
        user,
        tenant,
        token,
        isAuthenticated: true,
        isLoading: false,
        error: null,
      };

      set(newState);
      saveToStorage(newState);

      return { success: true };
    } catch (error: any) {
      update(state => ({
        ...state,
        isLoading: false,
        error: error.message || 'Login failed',
      }));
      return { success: false, error: error.message };
    }
  }

  // Logout function
  function logout() {
    const newState: AuthState = {
      user: null,
      tenant: null,
      token: null,
      isAuthenticated: false,
      isLoading: false,
      error: null,
    };

    set(newState);

    if (browser) {
      localStorage.removeItem('metabase_auth');
    }
  }

  // Update user profile
  async function updateProfile(updates: Partial<User>) {
    const currentState = get();
    if (!currentState.user || !currentState.token) {
      return { success: false, error: 'Not authenticated' };
    }

    update(state => ({
      ...state,
      isLoading: true,
      error: null,
    }));

    try {
      // In a real implementation, this would call the API
      // const response = await apiClient.put('/users/me', updates, {
      //   headers: { Authorization: `Bearer ${currentState.token}` }
      // });

      const updatedUser = { ...currentState.user, ...updates };

      update(state => ({
        ...state,
        user: updatedUser,
        isLoading: false,
      }));

      saveToStorage(get());
      return { success: true };
    } catch (error: any) {
      update(state => ({
        ...state,
        isLoading: false,
        error: error.message || 'Profile update failed',
      }));
      return { success: false, error: error.message };
    }
  }

  // Refresh token
  async function refreshToken() {
    const currentState = get();
    if (!currentState.token) {
      return false;
    }

    try {
      // In a real implementation, this would call the API
      // const response = await apiClient.post('/auth/refresh', {
      //   token: currentState.token
      // });

      // For now, just return true (mock implementation)
      return true;
    } catch (error) {
      console.error('Token refresh failed:', error);
      logout();
      return false;
    }
  }

  // Check authentication status
  async function checkAuth() {
    const currentState = get();
    if (!currentState.token || !currentState.user) {
      return false;
    }

    try {
      // In a real implementation, this would validate the token
      // const response = await apiClient.get('/auth/me', {
      //   headers: { Authorization: `Bearer ${currentState.token}` }
      // });

      return true;
    } catch (error) {
      console.error('Auth check failed:', error);
      logout();
      return false;
    }
  }

  // Utility to get current state
  function get() {
    let currentState: AuthState;
    subscribe(state => currentState = state)();
    return currentState!;
  }

  // Initialize on client-side
  if (browser) {
    loadFromStorage();
  }

  return {
    subscribe,
    login,
    logout,
    updateProfile,
    refreshToken,
    checkAuth,
    get,
  };
}

export const auth = createAuthStore();

// Derived stores
export const user = derived(auth, $auth => $auth.user);
export const tenant = derived(auth, $auth => $auth.tenant);
export const isAuthenticated = derived(auth, $auth => $auth.isAuthenticated);
export const isLoading = derived(auth, $auth => $auth.isLoading);
export const error = derived(auth, $auth => $auth.error);

// Permission checking
export const hasPermission = derived(
  [user],
  ([$user]) => (permission: string) => {
    if (!$user) return false;
    if ($user.role === 'admin') return true;
    return $user.permissions.includes(permission);
  }
);

export const hasRole = derived(
  [user],
  ([$user]) => (role: string) => {
    if (!$user) return false;
    return $user.role === role;
  }
);

export const canAccessAdmin = derived(
  [user],
  ([$user]) => {
    if (!$user) return false;
    return $user.role === 'admin' || $user.permissions.includes('admin');
  }
);

// User display helpers
export const userDisplayName = derived(
  [user],
  ([$user]) => {
    if (!$user) return 'Guest';
    return $user.email || $user.username || 'Unknown User';
  }
);

export const userInitials = derived(
  [user],
  ([$user]) => {
    if (!$user) return 'G';
    const name = $user.email || $user.username || '';
    const parts = name.split(/[\s.@]+/);
    if (parts.length >= 2) {
      return (parts[0][0] + parts[1][0]).toUpperCase();
    }
    return name.substring(0, 2).toUpperCase();
  }
);