<?php

/*
 * This file is part of the Omed project.
 *
 * (c) Anthonius Munthi <https://itstoni.com>
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

declare(strict_types=1);

namespace Omed\Component\User\Tests\Manager;

use Doctrine\Persistence\ObjectManager;
use Doctrine\Persistence\ObjectRepository;
use Omed\Component\ORM\Testing\DatabaseTestTrait;
use Omed\Component\User\Manager\UserManager;
use Omed\Component\User\Manager\UserManagerInterface;
use Omed\Component\User\Model\User;
use Omed\Component\User\Tests\TestUser;
use Omed\Component\User\Util\CanonicalFieldsUpdater;
use Omed\Component\User\Util\PasswordUpdaterInterface;
use PHPUnit\Framework\MockObject\MockObject;
use PHPUnit\Framework\TestCase;

class TestUserManager extends UserManager
{

}

class UserManagerTest extends TestCase
{
    use DatabaseTestTrait;

    /**
     * @var UserManagerInterface
     */
    private $userManager;

    /**
     * @var MockObject
     */
    private $om;

    /**
     * @var MockObject
     */
    private $repository;

    /**
     * @var MockObject
     */
    private $passwordUpdater;

    /**
     * @var MockObject
     */
    private $canonicalFieldsUpdater;

    protected function setUp(): void
    {
        $passwordUpdater = $this->getMockBuilder(PasswordUpdaterInterface::class)->getMock();
        $canonicalFieldsUpdater = $this->getMockBuilder(CanonicalFieldsUpdater::class)
            ->disableOriginalConstructor()
            ->getMock();

        $this->repository = $this->createMock(ObjectRepository::class);

        $this->om = $this->createMock(ObjectManager::class);
        $this->om->expects($this->any())
            ->method('getRepository')
            ->with($this->equalTo(TestUser::class))
            ->willReturn($this->repository);

        $this->passwordUpdater = $passwordUpdater;
        $this->canonicalFieldsUpdater = $canonicalFieldsUpdater;
        $this->userManager = new TestUserManager($passwordUpdater, $canonicalFieldsUpdater, $this->om, TestUser::class);
    }

    public function testConstruct()
    {
        $userManager = $this->userManager;

        $this->assertSame($this->passwordUpdater, $userManager->getPasswordUpdater());
    }

    public function testCreate()
    {
        $manager = $this->userManager;
        $om = $this->om;

        $user = $manager->createUser();
        $user->setUsername('test')
            ->setEmail('test@test.com')
            ->setPlainPassword('password')
            ->setPassword('password');

        $om->expects($this->once())
            ->method('persist')
            ->with($user);
        $om->expects($this->once())
            ->method('flush');
        $manager->storeUser($user);
        $this->assertInstanceOf(User::class, $user);
    }

    public function testStoreUser()
    {
        $manager = $this->userManager;
        $om = $this->om;

        $user = new TestUser();
        $user
            ->setUsername('test')
            ->setEmail('test@test.com');
        $om->expects($this->once())
            ->method('persist')
            ->with($user);

        $om->expects($this->once())
            ->method('flush');

        $manager->storeUser($user);
    }

    /**
     * @depends testCreate
     */
    public function testFindUserBy()
    {
        $manager = $this->userManager;
        $repository = $this->repository;

        $user = new TestUser();
        $criteria = ['email' => 'test@test.com'];
        $repository->expects($this->once())
            ->method('findOneBy')
            ->with($criteria)
            ->willReturn($user);

        $found = $manager->findUserBy($criteria);

        $this->assertSame($user, $found);
    }
}
