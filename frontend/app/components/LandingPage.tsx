import { SignUpButton, SignInButton } from "@clerk/react-router";
import Button from "./ui/Button";
import { Card, CardContent } from "./ui/Card";
import { STORY_PHASES } from "~/lib/storyPhases";

function AuthButtons({ dark = false }: { dark?: boolean }) {
  return (
    <div className="flex flex-col sm:flex-row gap-3 justify-center">
      <SignUpButton>
        <Button size="lg">Create an account</Button>
      </SignUpButton>
      <SignInButton>
        <Button
          size="lg"
          variant={dark ? "secondary" : "outline"}
          className={dark ? "border-transparent" : undefined}
        >
          Sign in
        </Button>
      </SignInButton>
    </div>
  );
}

export function LandingPage() {
  return (
    <div className="flex-1">
      {/* Hero */}
      <section className="bg-gradient-to-b from-primary-50 to-slate-50">
        <div className="max-w-6xl mx-auto px-4 pt-20 pb-16 text-center">
          <p className="text-sm font-semibold uppercase tracking-wider text-primary-600 mb-4">
            For introductory language courses
          </p>
          <h1 className="text-4xl md:text-6xl font-bold tracking-tight text-slate-900">
            Learn languages through
            <span className="block text-primary-600">interactive stories</span>
          </h1>
          <p className="mt-6 text-lg md:text-xl text-slate-600 max-w-2xl mx-auto leading-relaxed">
            Each story starts with a video, then walks you through
            identification, translation, writing, and recall — all on the same
            text.
          </p>
          <div className="mt-10">
            <AuthButtons />
          </div>
        </div>
      </section>

      {/* Phases */}
      <section className="max-w-6xl mx-auto px-4 py-16">
        <h2 className="text-2xl md:text-3xl font-bold tracking-tight text-slate-900 text-center mb-10">
          How a story works
        </h2>
        <ol className="grid sm:grid-cols-2 lg:grid-cols-3 gap-6">
          {STORY_PHASES.map((phase, i) => (
            <li key={phase.title}>
              <Card className="h-full hover:shadow-md transition-shadow">
                <CardContent className="p-6 h-full flex flex-col">
                  <div className="flex items-center justify-between mb-5">
                    <div className="w-11 h-11 bg-primary-100 rounded-lg flex items-center justify-center">
                      <span
                        className="material-icons text-primary-600"
                        aria-hidden="true"
                      >
                        {phase.icon}
                      </span>
                    </div>
                    <span className="text-sm font-semibold text-slate-400 tabular-nums">
                      {String(i + 1).padStart(2, "0")}
                    </span>
                  </div>
                  <h3 className="text-lg font-semibold text-slate-900 mb-2">
                    {phase.title}
                  </h3>
                  <p className="text-slate-600 text-sm leading-relaxed">
                    {phase.body}
                  </p>
                </CardContent>
              </Card>
            </li>
          ))}
        </ol>
      </section>

      {/* CTA */}
      <section className="bg-slate-900 text-white">
        <div className="max-w-3xl mx-auto px-4 py-20 text-center">
          <h2 className="text-3xl md:text-4xl font-bold tracking-tight mb-5">
            Ready to start?
          </h2>
          <p className="text-lg text-slate-300 mb-10 leading-relaxed">
            Glossias is designed to be used as part of a preexisting
            second-language course, not as a standalone project. If you are part
            of such a course, please create an account and your instructor will
            enroll you, or reach out to{" "}
            <a
              href="mailto:help@glossias.org"
              className="text-white underline underline-offset-2 hover:text-primary-200"
            >
              help@glossias.org
            </a>{" "}
            to be placed into a preview course.
          </p>
          <AuthButtons dark />
        </div>
      </section>
    </div>
  );
}
