
Resumebank.bizWebsite Prompt

****Overview****

You are a recruiting and web
development expert using the Go programming language. Using Go, a Go
template engine of your choice, the latest version, Bootstrap,
PostgreSQL and a WYSIWYG or rich text editor. Select an appropriate
database for the embeddings.  Your goal is to create a candidate
centric website that will allow job seekers to register with the site
and create a profile page that will include a formatted resume. The
resume will be able to be saved as formatted html that will be
displayed on the candidates profile page. Please allow the candidate
to use an appropriate text editor for this purpose. Please also
ensure the site is optimized for SEO on all pages included candidate,
company, and job pages. This will also include a sitemap that is
updated daily and a site search index that is updated hourly. The
site logo should be in the upper left hand corner.

Candidates should only be able to edit
their profiles.

Build a search function that will index
both the candidates and jobs. Use a check box to select between job
search and candidate search.

**Candidate Matching function** –
for employers using a local LLM RAG, with the active resume corpus as
the RAG truth source, employers paste in a job description and
receive a list of potential candidates. This list should include
links to candidate profiles. Indicate that this is feature is in
development and will get better with time. Please select an
appropriate text embedding model for this task.

**Job Matching function** – for
candidates using a local LLM RAG, with the active company jobs corpus
as the RAG truth source. Candidates can paste in there resume and
find the best matching jobs. This list should include links to the
relevant jobs. Indicate that this is feature is in development and
will get better with time. Please select an appropriate text
embedding model for this task.

****Candidate** **

The candidate
section should allow candidates to create a profile that will feed
into a resume or profile page that will be searchable on the website.
Display the results on a search results page.

*****Candidate Fields*****

Name – required

Title – field help = the title you
want prospective employers to see.

City

State

Zipcode - required

Email – required

Linked In URL

Skills – field help = enter skills
and keywords you want to ensure are indexed with your profile.

Candidate Summary -

Resume – WYSIWYG or rich text editor
interface – required

*****Candidate Profile Page*****

The candidate profile page should
include the candidate fields, formatted for SEO and place to upload a
pdf resume and three additional files. All files are public and
available for download.

****Employer** **

The employer
section is a where organizations can create a page to show case their
company and show the jobs they post in their jobs section. There
should also be a place for them to describe their organization and
state why people would want to work there.

*****Employer Fields*****

Company_Name –
required

Industry

City

State

Zipcode –
required

Phone – Optional

Email_Address –
Optional

Website

Description

Locations

*****Employer Profile*****

Please display the information in their
fields.

*****Job Posting Fields*****

Company_Name –
required

Date_posted

Title

Location

Job_Number –
from company

Salary_Min –
optional

Salary_Max –
optional

Description -
WYSIWYG or rich text editor interface – required

****Interaction****

Setup a way employers and candidates
through messages. Also setup a thumbs up or down on jobs only by
candidates.

Provide testing where possible to
ensure the website functions correctly. Run the tests.

Please ask questions if you need
additional information.

Provide information on how to run the
project locally.

Also provide information on deployment
options and procedures, pretend I am a complete novice.
