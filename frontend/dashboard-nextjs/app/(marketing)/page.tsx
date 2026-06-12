import Link from 'next/link';
import { cookies } from 'next/headers';
import { 
  ArrowRight, 
  Terminal, 
  Activity, 
  Database, 
  ShieldCheck, 
  Zap, 
  BookOpen, 
  Code2, 
  Box, 
  LayoutDashboard
} from 'lucide-react';

export default function LandingPage() {
  const cookieStore = cookies();
  const isAuthenticated = cookieStore.has('token');

  return (
    <div className="min-h-screen bg-slate-50 text-slate-900 font-sans selection:bg-blue-200">
      
      {/* Header */}
      <header className="sticky top-0 z-50 w-full border-b border-slate-200 bg-white/80 backdrop-blur-md">
        <div className="container mx-auto px-4 h-16 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="bg-blue-600 p-1.5 rounded-lg">
              <Activity className="w-5 h-5 text-white" />
            </div>
            <span className="font-bold text-xl tracking-tight text-slate-900">IICPC</span>
          </div>
          <nav className="hidden md:flex items-center gap-8 text-sm font-medium text-slate-600">
            <Link href="#features" className="hover:text-blue-600 transition-colors">Features</Link>
            <Link href="#guide" className="hover:text-blue-600 transition-colors">Submission Guide</Link>
            <Link href="/dashboard" className="hover:text-blue-600 transition-colors">Dashboard</Link>
          </nav>
          <div className="flex items-center gap-4">
            {!isAuthenticated ? (
              <>
                <Link href="/auth/login" className="text-sm font-medium text-slate-600 hover:text-blue-600 transition-colors">
                  Log in
                </Link>
                <Link href="/auth/register" className="text-sm font-medium bg-blue-600 text-white px-4 py-2 rounded-full hover:bg-blue-700 transition-all shadow-sm shadow-blue-200 hover:shadow-blue-300">
                  Get Started
                </Link>
              </>
            ) : (
              <Link href="/dashboard" className="text-sm font-medium bg-blue-600 text-white px-4 py-2 rounded-full hover:bg-blue-700 transition-all shadow-sm shadow-blue-200 hover:shadow-blue-300">
                Go to Dashboard
              </Link>
            )}
          </div>
        </div>
      </header>

      {/* Hero Section */}
      <section className="relative pt-24 pb-32 overflow-hidden">
        {/* Background Decorative Elements */}
        <div className="absolute top-0 inset-x-0 h-full overflow-hidden pointer-events-none -z-10">
          <div className="absolute -top-40 -right-40 w-[800px] h-[800px] bg-blue-100/50 rounded-full blur-3xl opacity-60"></div>
          <div className="absolute top-40 -left-20 w-[600px] h-[600px] bg-indigo-100/50 rounded-full blur-3xl opacity-60"></div>
        </div>

        <div className="container mx-auto px-4 text-center">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-blue-50 border border-blue-100 text-blue-700 text-sm font-medium mb-8 animate-in fade-in slide-in-from-bottom-4 duration-700">
            <span className="flex h-2 w-2 rounded-full bg-blue-500 animate-pulse"></span>
            IICPC Distributed Benchmarking Platform
          </div>
          
          <h1 className="text-5xl md:text-7xl font-extrabold tracking-tight text-slate-900 mb-6 max-w-4xl mx-auto animate-in fade-in slide-in-from-bottom-6 duration-700 delay-100">
            Evaluate your trading bots at <span className="text-transparent bg-clip-text bg-gradient-to-r from-blue-600 to-indigo-600">hyperscale.</span>
          </h1>
          
          <p className="text-xl text-slate-600 mb-10 max-w-2xl mx-auto leading-relaxed animate-in fade-in slide-in-from-bottom-8 duration-700 delay-200">
            A high-performance, distributed infrastructure designed to validate, benchmark, and track latency for algorithmic trading submissions in real-time.
          </p>
          
          <div className="flex flex-col sm:flex-row items-center justify-center gap-4 animate-in fade-in slide-in-from-bottom-10 duration-700 delay-300">
            {!isAuthenticated ? (
              <Link href="/auth/register" className="w-full sm:w-auto inline-flex justify-center items-center gap-2 bg-blue-600 text-white px-8 py-4 rounded-full font-semibold text-lg hover:bg-blue-700 transition-all shadow-lg shadow-blue-600/20 hover:shadow-blue-600/30 hover:-translate-y-0.5">
                Start Benchmarking <ArrowRight className="w-5 h-5" />
              </Link>
            ) : (
              <Link href="/dashboard" className="w-full sm:w-auto inline-flex justify-center items-center gap-2 bg-blue-600 text-white px-8 py-4 rounded-full font-semibold text-lg hover:bg-blue-700 transition-all shadow-lg shadow-blue-600/20 hover:shadow-blue-600/30 hover:-translate-y-0.5">
                Go to Dashboard <ArrowRight className="w-5 h-5" />
              </Link>
            )}
            <Link href="#guide" className="w-full sm:w-auto inline-flex justify-center items-center gap-2 bg-white text-slate-700 px-8 py-4 rounded-full font-semibold text-lg border border-slate-200 hover:bg-slate-50 hover:border-slate-300 transition-all hover:-translate-y-0.5 shadow-sm">
              <BookOpen className="w-5 h-5 text-slate-400" /> View Documentation
            </Link>
          </div>
        </div>
      </section>

      {/* Bento Grid Features Section */}
      <section id="features" className="py-24 bg-white border-y border-slate-100 relative">
        <div className="container mx-auto px-4">
          <div className="mb-16 text-center">
            <h2 className="text-3xl md:text-4xl font-bold tracking-tight text-slate-900 mb-4">Built for extreme performance</h2>
            <p className="text-slate-600 max-w-2xl mx-auto">Everything you need to validate your strategies against a simulated high-throughput exchange environment.</p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-4 gap-6 max-w-6xl mx-auto auto-rows-[240px]">
            {/* Feature 1 (Large) */}
            <div className="md:col-span-2 md:row-span-2 rounded-3xl bg-gradient-to-br from-slate-50 to-blue-50/50 border border-slate-200 p-8 flex flex-col relative overflow-hidden group hover:border-blue-200 transition-colors">
              <div className="absolute top-0 right-0 p-8 opacity-10 group-hover:opacity-20 transition-opacity">
                <Activity className="w-32 h-32 text-blue-600" />
              </div>
              <div className="bg-blue-100 text-blue-700 w-12 h-12 rounded-xl flex items-center justify-center mb-6">
                <Zap className="w-6 h-6" />
              </div>
              <h3 className="text-2xl font-bold text-slate-900 mb-3 z-10">Real-time Telemetry</h3>
              <p className="text-slate-600 leading-relaxed z-10 max-w-md">
                Monitor your bot's performance live. Our TimescaleDB-backed pipeline captures TPS, p50/p90/p99 latencies, and resource usage with millisecond precision, streamed directly to your dashboard via WebSockets.
              </p>
            </div>

            {/* Feature 2 (Medium) */}
            <div className="md:col-span-2 rounded-3xl bg-white border border-slate-200 p-8 flex flex-col group hover:shadow-md hover:shadow-slate-200/50 transition-all">
              <div className="flex items-center gap-4 mb-4">
                <div className="bg-indigo-100 text-indigo-700 w-10 h-10 rounded-lg flex items-center justify-center">
                  <ShieldCheck className="w-5 h-5" />
                </div>
                <h3 className="text-xl font-bold text-slate-900">Secure Validation</h3>
              </div>
              <p className="text-slate-600 leading-relaxed">
                Every submission is strictly validated in an isolated sandbox. We check compilation, static analysis, and basic runtime safety before your code ever touches the benchmarking fleet.
              </p>
            </div>

            {/* Feature 3 (Small) */}
            <div className="rounded-3xl bg-slate-900 text-white border border-slate-800 p-8 flex flex-col relative overflow-hidden">
               <div className="absolute -bottom-4 -right-4 text-slate-800">
                <Terminal className="w-24 h-24" />
              </div>
              <h3 className="text-lg font-bold mb-2 z-10">Multi-Language</h3>
              <p className="text-slate-400 text-sm z-10">Support for C++, Rust, Go, Python, and more via standardized Docker containers.</p>
            </div>

            {/* Feature 4 (Small) */}
            <div className="rounded-3xl bg-blue-600 text-white border border-blue-500 p-8 flex flex-col">
              <h3 className="text-lg font-bold mb-2">Automated Ops</h3>
              <p className="text-blue-100 text-sm">Submit your code and our orchestrator handles the building, deployment, and teardown entirely.</p>
            </div>
            
            {/* Feature 5 (Medium) */}
            <div className="md:col-span-2 rounded-3xl bg-white border border-slate-200 p-8 flex flex-col group hover:shadow-md hover:shadow-slate-200/50 transition-all">
              <div className="flex items-center gap-4 mb-4">
                <div className="bg-emerald-100 text-emerald-700 w-10 h-10 rounded-lg flex items-center justify-center">
                  <Database className="w-5 h-5" />
                </div>
                <h3 className="text-xl font-bold text-slate-900">Redpanda Event Streaming</h3>
              </div>
              <p className="text-slate-600 leading-relaxed">
                Powered by Redpanda (Kafka-compatible) to handle millions of orders and market data events per second without dropping a single packet.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* Submission Guide Section */}
      <section id="guide" className="py-24 bg-slate-50">
        <div className="container mx-auto px-4">
          <div className="max-w-4xl mx-auto">
            <div className="text-center mb-16">
              <div className="inline-flex items-center justify-center bg-blue-100 text-blue-700 w-16 h-16 rounded-2xl mb-6">
                <Code2 className="w-8 h-8" />
              </div>
              <h2 className="text-3xl md:text-4xl font-bold tracking-tight text-slate-900 mb-4">Submission Guide</h2>
              <p className="text-slate-600">Follow these steps to package and deploy your trading bot to the platform.</p>
            </div>

            <div className="space-y-8 relative before:absolute before:inset-0 before:ml-5 before:-translate-x-px md:before:mx-auto md:before:translate-x-0 before:h-full before:w-0.5 before:bg-gradient-to-b before:from-transparent before:via-slate-200 before:to-transparent">
              
              {/* Step 1 */}
              <div className="relative flex items-center justify-between md:justify-normal md:odd:flex-row-reverse group is-active">
                <div className="flex items-center justify-center w-10 h-10 rounded-full border-4 border-slate-50 bg-blue-600 text-white font-bold shrink-0 md:order-1 md:group-odd:-translate-x-1/2 md:group-even:translate-x-1/2 shadow-md">
                  1
                </div>
                <div className="w-[calc(100%-4rem)] md:w-[calc(50%-2.5rem)] p-6 rounded-2xl bg-white border border-slate-200 shadow-sm">
                  <h3 className="font-bold text-slate-900 text-lg mb-2 flex items-center gap-2">
                    <Box className="w-5 h-5 text-blue-500" /> Structure your code
                  </h3>
                  <p className="text-slate-600 text-sm">
                    Ensure your bot connects to the exchange via the environment variables <code>MARKET_DATA_URL</code> and <code>ORDER_ENTRY_URL</code>. Write your logic to consume the feed and submit orders.
                  </p>
                </div>
              </div>

              {/* Step 2 */}
              <div className="relative flex items-center justify-between md:justify-normal md:odd:flex-row-reverse group is-active">
                <div className="flex items-center justify-center w-10 h-10 rounded-full border-4 border-slate-50 bg-blue-600 text-white font-bold shrink-0 md:order-1 md:group-odd:-translate-x-1/2 md:group-even:translate-x-1/2 shadow-md">
                  2
                </div>
                <div className="w-[calc(100%-4rem)] md:w-[calc(50%-2.5rem)] p-6 rounded-2xl bg-white border border-slate-200 shadow-sm">
                  <h3 className="font-bold text-slate-900 text-lg mb-2 flex items-center gap-2">
                    <Terminal className="w-5 h-5 text-blue-500" /> Write a Dockerfile
                  </h3>
                  <p className="text-slate-600 text-sm">
                    Create a <code>Dockerfile</code> at the root of your project. It should build your code and define the `ENTRYPOINT` to start your bot. The orchestrator will use this to build your image.
                  </p>
                  <pre className="mt-3 bg-slate-900 text-slate-300 p-3 rounded-lg text-xs overflow-x-auto">
                    <code>
{`FROM python:3.11-slim
WORKDIR /app
COPY . .
RUN pip install -r requirements.txt
ENTRYPOINT ["python", "main.py"]`}
                    </code>
                  </pre>
                </div>
              </div>

              {/* Step 3 */}
              <div className="relative flex items-center justify-between md:justify-normal md:odd:flex-row-reverse group is-active">
                <div className="flex items-center justify-center w-10 h-10 rounded-full border-4 border-slate-50 bg-blue-600 text-white font-bold shrink-0 md:order-1 md:group-odd:-translate-x-1/2 md:group-even:translate-x-1/2 shadow-md">
                  3
                </div>
                <div className="w-[calc(100%-4rem)] md:w-[calc(50%-2.5rem)] p-6 rounded-2xl bg-white border border-slate-200 shadow-sm">
                  <h3 className="font-bold text-slate-900 text-lg mb-2 flex items-center gap-2">
                    <LayoutDashboard className="w-5 h-5 text-blue-500" /> Upload via Dashboard
                  </h3>
                  <p className="text-slate-600 text-sm">
                    Zip your project directory (including the Dockerfile). Log in to the platform, navigate to the Dashboard, and upload your zip file. The system will automatically validate, build, and benchmark your code.
                  </p>
                </div>
              </div>

            </div>
            
            <div className="mt-12 text-center">
              {!isAuthenticated ? (
                <Link href="/auth/register" className="inline-flex items-center justify-center bg-slate-900 text-white px-6 py-3 rounded-full font-medium hover:bg-slate-800 transition-colors">
                  Ready? Create an account
                </Link>
              ) : (
                <Link href="/dashboard" className="inline-flex items-center justify-center bg-slate-900 text-white px-6 py-3 rounded-full font-medium hover:bg-slate-800 transition-colors">
                  Go to Dashboard
                </Link>
              )}
            </div>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="bg-white border-t border-slate-200 py-12">
        <div className="container mx-auto px-4 flex flex-col md:flex-row items-center justify-between gap-4">
          <div className="flex items-center gap-2">
            <Activity className="w-5 h-5 text-blue-600" />
            <span className="font-bold text-slate-900">IICPC Benchmarks</span>
          </div>
          <p className="text-slate-500 text-sm">
            © {new Date().getFullYear()} IICPC. All rights reserved.
          </p>
        </div>
      </footer>
    </div>
  );
}
