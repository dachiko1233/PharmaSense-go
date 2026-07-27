import type { MetadataRoute } from "next";

export default function robots(): MetadataRoute.Robots {
  return {
    rules: {
      userAgent: "*",
      disallow: [
        "/*/dashboard",
        "/*/inventory",
        "/*/alerts",
        "/*/reports",
        "/*/import",
        "/*/chain",
        "/*/billing",
        "/*/settings",
        "/*/login",
        "/*/signup",
        "/*/forgot-password",
        "/*/reset-password",
        "/*/verify-email",
      ],
    },
  };
}
