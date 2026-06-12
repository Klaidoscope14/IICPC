import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

export function middleware(request: NextRequest) {
  // Check for the authentication cookie
  const token = request.cookies.get('token')?.value;

  // Paths that require authentication
  const protectedPaths = ['/dashboard', '/submissions', '/leaderboard', '/upload', '/benchmarks', '/monitor'];
  const isProtectedPath = protectedPaths.some((path) => request.nextUrl.pathname.startsWith(path));

  if (isProtectedPath && !token) {
    // Redirect to login if unauthenticated user tries to access protected route
    const loginUrl = new URL('/auth/login', request.url);
    return NextResponse.redirect(loginUrl);
  }

  // Paths for guests only (e.g., login, register)
  const guestPaths = ['/auth/login', '/auth/register'];
  const isGuestPath = guestPaths.some((path) => request.nextUrl.pathname.startsWith(path));

  if (isGuestPath && token) {
    // Redirect to dashboard if authenticated user tries to access login/register
    const dashboardUrl = new URL('/dashboard', request.url);
    return NextResponse.redirect(dashboardUrl);
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    '/dashboard/:path*',
    '/submissions/:path*',
    '/leaderboard/:path*',
    '/upload/:path*',
    '/benchmarks/:path*',
    '/monitor/:path*',
    '/auth/login',
    '/auth/register',
  ],
};
