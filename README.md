# Sheep Game

The object of the game is for players to try and guess the most popular answer to a general question.  An example
question might be: "someone famous with the first name George".  The players must try and guess the answer which the 
majority of the other players will answer here.  Example answers might be "George Washington", "George Clooney", etc.
Points are awarded based on the number of people who picked the same answer.  Questions should not have a "correct"
answer.  For example, the question, "The height of the tallest building in New York" should not be used -- it has some
correct answer.  Rather, it would be more appropriate to use, "The name of a tall building in New York" -- a question
with multiple possible answers, none of them "correct".

The Sheep game can be run in teams or individual mode.  In individual mode, no master list of players is required -- 
this makes it easier to get started.  If running in teams mode, a teams file must be supplied to enumerate the teams 
and their members.  The teams file must be a JSON file formatted as follows:

```json
{
  "TeamA": [
    {
      "email": "jjc@acme.com",
      "name": "Jim Croche"
    },
    {
      "email": "bhf@acme.com",
      "name": "Bill Fettman"
    }
  ],
  "TeamB": [
    {
      "email": "abc@acme.com",
      "name": "Allen Crab"
    },
    {
      "email": "mrb@acme.com",
      "name": "Mike Brown"
    }
  ],
  "TeamC": [
    {
      "email": "xxx@acme.com",
      "name": "XXX"
    },
    {
      "email": "jdoe@acme.com",
      "name": "John Doe"
    }
  ]
}
```

Teams mode also introduces more complex scoring where the team gets a score composed of the total of all the scores
of the members.  If a member does not submit a set of answers to a quiz, the score of the missing member should 
not be left at zero as that would doom the team to failure and the whole point of the game is to provide fun, 
comradiery, and interaction within the organization (failure isn't fun).  There are three ways a missing member
can be scored with this program: average, least, and middle.  Average takes the average of all the scores of those
who submitted answers, least picks the lowest score, and middle picks the midpoint or median score (average is the 
default mode unless another is choosen).

You should strive to have the same number of members on each team.  This is hard to maintain over time, but it does
make it more fun.  Three to six is a good number.  It is possible to rearrange teams at any point simply by adjusting
the JSON file.  If you do change teams, it is recommended to keep a history of the JSON teams files along with the 
quiz results file so that you can always go back and rescore a quiz as it was at the time.  This is useful if you
want to total scores over time instead of or an addition to a single quiz scoring.  For example, at the end of a year,
players or teams can be scored for all-time high scores, total high scores, all-time low scores, etc.

## Answer Normalization

The hardest part of running the quiz, besides coming up with the questions, is the answer normalization.  
This is a manual process whereby all similar answers must be grouped so that all similar answers become the same
exact text.  I find this is easiest to do while the answers are still in spreadsheet format.  I sort the rows
by the answers in question #1. Then I look for commonalities in the answers, highlight all those common answers and
execute a Copy-Down function (CTRL-D in Excel or Meta-D in Google Sheets).  Sometimes I have to remove "the" or "a" or
other common words in front of some answers.

Players might try and make social, political, or company cultural comments which often are unacceptable or which are 
not appropriate for the Quiz.  Someone may feel like they are being singled out or bullied by an answer.  Someone 
might have tried to make a joke but their humor might not translate to everyone in a safe manner.  The quiz master
must look for all these cases and deal with the answers.  In some cases, the answer can be sanitized and in others
just removed entirely (leaving a blank answer for that player).

Finally, there will always be a subset of players who are trying to be funny with their answer.  If the humor seems
harmless, I'll just leave the answer in place knowing they will obviously get a low score.  Other times, the
player obviously just is too lazy to answer properly and in that case, I might, again, just leave the answer knowing
they are going to get a low score.

## Answer presentation

During answer presentation, you might find that you missed an answer normalization,  For example, let's say we had
the question, "A sweetener for food or drink".  Maybe we got the following answers:
    Saccrrin
    Sweet and Low
    Sugar
    Steevia
    Fructose
    Sweet-n-Low
    Sacirine
    Sucrose

You normlized the answers to:
    Saccrrin
    Sweet and Low
    Sugar
    Steevia
    Fructose
    Sucrose

During the answer presentation, someone points out that Sweet and Low is the same thing as Saccrrin and you realize 
that you missed that.  You're the judge, you can choose to overrule the objection or, you can correct it.  If you 
wish to correct it, you would just edit the answers and rerun the tabulation again.

## Bonus Questions

You can play with bonus values for certain questions.  Anyone getting the bonus answer will get an extra
score equal to 150% of the top answer for the question.  For example, if the answers were as follows:

    10 Chocolate
     5 Vanilla
     1 Strawberry

and Strawberry was the bonus answer, then the user(s) who answered Strawberry would get a score of 16 for that question;
as opposed to 1 in the above example.  If the bonus answer were Chocolate, then 10 people would get a score of 26
for the question.  Obviously, the bonus feature is designed for a non-common answer.  Maybe the quiz master wants
to emphasize an answer.  For example, in the above example, maybe people know the quiz master has a Miss Strawberry
character in the background of their web cam and so players might be able to put two and two together to score the
bonus.

You can mark a question as a bonus question by prepending the question with the 🎯 character.  If the question
starts with the 🎯 character, you must also append in square brackets the answer which qualifies for the bonus.
For example:

    🎯 A song by the group Cold Play [Adventure Of A Lifetime]

In this example, any answers consisting of "adventure of a lifetime" will get a bonus score added.   

## Microsoft Forms

## Google Forms

When using Google forms, you should create a _______ type of form.  You should restrict the end-time of when
submissions must be supplied.  You must collect the email of the players.  Google Forms will not supply a name with the
email address.  There is not a good way to get the names from email addresses using the Google APIs (the API to do
that is helplessly broken at this point).

When scoring the quiz, you export the Google Forms result as a Google Sheet and then download the sheet as a CSV file.


