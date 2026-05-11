"use client";

import { create } from "zustand";

interface User {
  id: string;
  email: string;
  full_name: string;
  email_verified: boolean;
  sms_enabled: boolean;
  default_pharmacy_id: string;
}

interface Pharmacy {
  id: string;
  name: string;
  license_number: string;
  plan: string;
  language: string;
  subscription_status?: string;
}

interface AuthState {
  token: string | null;
  user: User | null;
  pharmacy: Pharmacy | null;
  role: string | null;
  isAuthenticated: boolean;
  setAuth: (token: string, user: User, pharmacy: Pharmacy, role?: string) => void;
  setPharmacy: (pharmacy: Pharmacy, token: string) => void;
  logout: () => void;
  hydrate: () => void;
}

export const useAuth = create<AuthState>((set) => ({
  token: null,
  user: null,
  pharmacy: null,
  role: null,
  isAuthenticated: false,

  setAuth: (token, user, pharmacy, role) => {
    localStorage.setItem("token", token);
    const parsedRole = role ?? parseRoleFromToken(token);
    set({ token, user, pharmacy, role: parsedRole, isAuthenticated: true });
  },

  setPharmacy: (pharmacy, token) => {
    localStorage.setItem("token", token);
    const parsedRole = parseRoleFromToken(token);
    set({ pharmacy, token, role: parsedRole });
  },

  logout: () => {
    localStorage.removeItem("token");
    set({ token: null, user: null, pharmacy: null, role: null, isAuthenticated: false });
  },

  hydrate: () => {
    const token = localStorage.getItem("token");
    if (!token) return;
    try {
      const payload = JSON.parse(atob(token.split(".")[1]));
      const now = Date.now() / 1000;
      if (payload.exp && payload.exp < now) {
        localStorage.removeItem("token");
        return;
      }
      set({
        token,
        isAuthenticated: true,
        role: payload.role,
      });
    } catch {
      localStorage.removeItem("token");
    }
  },
}));

function parseRoleFromToken(token: string): string {
  try {
    const payload = JSON.parse(atob(token.split(".")[1]));
    return payload.role ?? "staff";
  } catch {
    return "staff";
  }
}
