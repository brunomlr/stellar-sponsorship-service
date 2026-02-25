import NextAuth from "next-auth";
import Google from "next-auth/providers/google";

export const { handlers, auth, signIn, signOut } = NextAuth({
  providers: [
    Google({
      clientId: process.env.GOOGLE_CLIENT_ID!,
      clientSecret: process.env.GOOGLE_CLIENT_SECRET!,
      authorization: {
        params: {
          hd: process.env.GOOGLE_ALLOWED_DOMAIN,
          scope: "openid email profile",
          prompt: "select_account",
        },
      },
    }),
  ],
  callbacks: {
    authorized({ auth, request }) {
      const isLoggedIn = !!auth?.user;
      const isOnLogin = request.nextUrl.pathname.startsWith("/login");
      const isAuthRoute = request.nextUrl.pathname.startsWith("/api/auth");

      if (isAuthRoute) return true;
      if (isOnLogin) return true;
      if (!isLoggedIn) return false; // NextAuth will redirect to pages.signIn

      return true;
    },
    async jwt({ token, account }) {
      if (account) {
        token.id_token = account.id_token;
      }
      return token;
    },
    async session({ session, token }) {
      session.id_token = token.id_token as string;
      return session;
    },
    async signIn({ profile }) {
      const allowedDomain = process.env.GOOGLE_ALLOWED_DOMAIN;
      if (allowedDomain && profile?.hd !== allowedDomain) {
        return false;
      }
      return true;
    },
  },
  pages: {
    signIn: "/login",
    error: "/login",
  },
});
